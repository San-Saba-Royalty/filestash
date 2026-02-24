package plg_backend_azurefileshare

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/directory"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/file"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/service"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/share"

	. "github.com/mickael-kerjean/filestash/server/common"
)

// AzureFileShare implements IBackend for Azure File Share (SMB/REST).
type AzureFileShare struct {
	client *service.Client
	ctx    context.Context
}

func init() {
	Backend.Register("azurefileshare", &AzureFileShare{})
}

// Init creates a service.Client using SharedKey authentication.
func (this *AzureFileShare) Init(params map[string]string, app *App) (IBackend, error) {
	accountName := params["account_name"]
	accountKey := params["account_key"]
	if accountName == "" || accountKey == "" {
		return nil, ErrAuthenticationFailed
	}

	cred, err := service.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		Log.Debug("plg_backend_azurefileshare::cred_error %s", err.Error())
		return nil, ErrAuthenticationFailed
	}

	serviceURL := fmt.Sprintf("https://%s.file.core.windows.net/", accountName)
	client, err := service.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		Log.Debug("plg_backend_azurefileshare::client_error %s", err.Error())
		return nil, ErrAuthenticationFailed
	}

	return &AzureFileShare{
		client: client,
		ctx:    app.Context,
	}, nil
}

func (this *AzureFileShare) LoginForm() Form {
	return Form{
		Elmnts: []FormElement{
			{Name: "type", Type: "hidden", Value: "azurefileshare"},
			{Name: "account_name", Type: "text", Placeholder: "Account Name"},
			{Name: "account_key", Type: "password", Placeholder: "Account Key"},
			{Name: "path", Type: "text", Placeholder: "Path"},
		},
	}
}

// afsPath splits a filestash path into share name and file/directory path.
type afsPath struct {
	shareName string // e.g. "document-templates"
	filePath  string // e.g. "folder/file.docx" (no leading slash, no trailing slash)
}

func parsePath(path string) afsPath {
	path = strings.TrimLeft(path, "/")
	idx := strings.IndexByte(path, '/')
	if idx < 0 {
		return afsPath{shareName: path}
	}
	return afsPath{
		shareName: path[:idx],
		filePath:  strings.TrimRight(path[idx+1:], "/"),
	}
}

// shareClient returns a share.Client for the given share name.
func (this *AzureFileShare) shareClient(shareName string) *share.Client {
	return this.client.NewShareClient(shareName)
}

// dirClient returns the directory.Client for dirPath within the share.
// An empty dirPath gives the root directory client.
func (this *AzureFileShare) dirClient(sc *share.Client, dirPath string) *directory.Client {
	if dirPath == "" {
		return sc.NewRootDirectoryClient()
	}
	return sc.NewDirectoryClient(dirPath)
}

// fileClient returns the file.Client for filePath within the share.
// The filePath must be the full path within the share (e.g. "dir/subdir/file.txt").
func (this *AzureFileShare) fileClient(sc *share.Client, filePath string) *file.Client {
	// We need to split into the parent directory and the file name.
	parent := filepath.Dir(filePath)
	if parent == "." {
		parent = ""
	}
	return this.dirClient(sc, parent).NewFileClient(filepath.Base(filePath))
}

// Ls lists files and directories at a given path.
func (this *AzureFileShare) Ls(path string) ([]os.FileInfo, error) {
	files := make([]os.FileInfo, 0)
	ap := parsePath(path)

	// Root: list all shares under the storage account.
	if ap.shareName == "" {
		pager := this.client.NewListSharesPager(nil)
		for pager.More() {
			resp, err := pager.NextPage(this.ctx)
			if err != nil {
				return files, err
			}
			for _, s := range resp.Shares {
				if s.Name == nil {
					continue
				}
				var mtime int64 = -1
				if s.Properties != nil && s.Properties.LastModified != nil {
					mtime = s.Properties.LastModified.Unix()
				}
				files = append(files, File{
					FName: *s.Name,
					FType: "directory",
					FTime: mtime,
					FSize: -1,
				})
			}
		}
		return files, nil
	}

	// Inside a share: list files and directories.
	sc := this.shareClient(ap.shareName)
	dc := this.dirClient(sc, ap.filePath)
	pager := dc.NewListFilesAndDirectoriesPager(nil)

	for pager.More() {
		resp, err := pager.NextPage(this.ctx)
		if err != nil {
			return files, err
		}
		if resp.Segment == nil {
			continue
		}
		for _, d := range resp.Segment.Directories {
			if d.Name == nil {
				continue
			}
			files = append(files, File{
				FName: *d.Name,
				FType: "directory",
				FTime: -1,
				FSize: -1,
			})
		}
		for _, f := range resp.Segment.Files {
			if f.Name == nil {
				continue
			}
			var size int64 = -1
			if f.Properties != nil && f.Properties.ContentLength != nil {
				size = *f.Properties.ContentLength
			}
			files = append(files, File{
				FName: *f.Name,
				FType: "file",
				FTime: -1,
				FSize: size,
			})
		}
	}
	return files, nil
}

// Stat returns metadata about a file or directory.
func (this *AzureFileShare) Stat(path string) (os.FileInfo, error) {
	ap := parsePath(path)
	if ap.shareName == "" || ap.filePath == "" {
		return File{
			FName: filepath.Base(path),
			FType: "directory",
			FTime: -1,
		}, nil
	}

	sc := this.shareClient(ap.shareName)
	fc := this.fileClient(sc, ap.filePath)
	props, err := fc.GetProperties(this.ctx, nil)
	if err != nil {
		// Assume it's a directory.
		return File{
			FName: filepath.Base(path),
			FType: "directory",
			FTime: -1,
		}, nil
	}

	var size int64
	if props.ContentLength != nil {
		size = *props.ContentLength
	}
	var mtime int64 = -1
	if props.LastModified != nil {
		mtime = props.LastModified.Unix()
	}

	return File{
		FName: filepath.Base(path),
		FType: "file",
		FSize: size,
		FTime: mtime,
	}, nil
}

// Cat downloads a file and returns a ReadCloser of its contents.
func (this *AzureFileShare) Cat(path string) (io.ReadCloser, error) {
	ap := parsePath(path)
	if ap.shareName == "" || ap.filePath == "" {
		return nil, ErrNotValid
	}

	sc := this.shareClient(ap.shareName)
	fc := this.fileClient(sc, ap.filePath)
	resp, err := fc.DownloadStream(this.ctx, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// Mkdir creates a directory (or a share at the root level).
func (this *AzureFileShare) Mkdir(path string) error {
	ap := parsePath(path)
	if ap.shareName == "" {
		return ErrNotValid
	}

	sc := this.shareClient(ap.shareName)
	if ap.filePath == "" {
		_, err := sc.Create(this.ctx, nil)
		return err
	}

	dc := this.dirClient(sc, ap.filePath)
	_, err := dc.Create(this.ctx, nil)
	return err
}

// Rm deletes a file or recursively deletes a directory.
func (this *AzureFileShare) Rm(path string) error {
	ap := parsePath(path)
	if ap.shareName == "" {
		return ErrNotValid
	}

	sc := this.shareClient(ap.shareName)
	if ap.filePath == "" {
		_, err := sc.Delete(this.ctx, nil)
		return err
	}

	// If not a directory path, try to delete as file first.
	if !strings.HasSuffix(path, "/") {
		fc := this.fileClient(sc, ap.filePath)
		if _, err := fc.Delete(this.ctx, nil); err == nil {
			return nil
		}
	}

	// Delete directory recursively.
	return rmDirRecursive(this.ctx, sc, ap.filePath)
}

// rmDirRecursive deletes all contents of a directory and then the directory itself.
func rmDirRecursive(ctx context.Context, sc *share.Client, dirPath string) error {
	dc := sc.NewDirectoryClient(dirPath)
	pager := dc.NewListFilesAndDirectoriesPager(nil)

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}
		if resp.Segment == nil {
			continue
		}
		for _, f := range resp.Segment.Files {
			if f.Name == nil {
				continue
			}
			childPath := dirPath + "/" + *f.Name
			childDir := filepath.Dir(childPath)
			if childDir == "." {
				childDir = ""
			}
			fc := sc.NewDirectoryClient(childDir).NewFileClient(*f.Name)
			if _, err := fc.Delete(ctx, nil); err != nil {
				return err
			}
		}
		for _, d := range resp.Segment.Directories {
			if d.Name == nil {
				continue
			}
			subDirPath := dirPath + "/" + *d.Name
			if err := rmDirRecursive(ctx, sc, subDirPath); err != nil {
				return err
			}
		}
	}

	_, err := dc.Delete(ctx, nil)
	return err
}

// Mv renames or moves a file using the Azure File Share rename API.
func (this *AzureFileShare) Mv(from string, to string) error {
	if from == to {
		return nil
	}
	afrom := parsePath(from)
	ato := parsePath(to)

	if afrom.shareName == "" || afrom.filePath == "" {
		return ErrNotSupported
	}
	if afrom.shareName != ato.shareName {
		// Cross-share moves are not supported.
		return ErrNotSupported
	}

	sc := this.shareClient(afrom.shareName)
	fc := this.fileClient(sc, afrom.filePath)
	_, err := fc.Rename(this.ctx, ato.filePath, nil)
	return err
}

// Touch creates an empty 0-byte file.
func (this *AzureFileShare) Touch(path string) error {
	ap := parsePath(path)
	if ap.shareName == "" || ap.filePath == "" {
		return ErrNotValid
	}

	sc := this.shareClient(ap.shareName)
	fc := this.fileClient(sc, ap.filePath)
	_, err := fc.Create(this.ctx, 0, nil)
	return err
}

// Save uploads content from reader to the given path.
func (this *AzureFileShare) Save(path string, reader io.Reader) error {
	ap := parsePath(path)
	if ap.shareName == "" || ap.filePath == "" {
		return ErrNotValid
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	size := int64(len(data))

	sc := this.shareClient(ap.shareName)
	fc := this.fileClient(sc, ap.filePath)

	// First create/resize the file to the correct size.
	if _, err := fc.Create(this.ctx, size, nil); err != nil {
		return err
	}

	if size == 0 {
		return nil
	}

	_, err = fc.UploadRange(this.ctx, 0, &readSeekCloser{data: data}, nil)
	return err
}

// Meta returns metadata constraints for path.
func (this *AzureFileShare) Meta(path string) Metadata {
	if path == "/" {
		return Metadata{
			CanCreateFile: NewBool(false),
			CanRename:     NewBool(false),
			CanMove:       NewBool(false),
			CanUpload:     NewBool(false),
		}
	}
	return Metadata{
		CanMove: NewBool(false),
	}
}

// readSeekCloser wraps a byte slice to implement io.ReadSeekCloser for UploadRange.
type readSeekCloser struct {
	data   []byte
	offset int64
}

func (r *readSeekCloser) Read(p []byte) (int, error) {
	if r.offset >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.offset:])
	r.offset += int64(n)
	return n, nil
}

func (r *readSeekCloser) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = r.offset + offset
	case io.SeekEnd:
		newOffset = int64(len(r.data)) + offset
	default:
		return r.offset, os.ErrInvalid
	}
	if newOffset < 0 {
		return r.offset, os.ErrInvalid
	}
	r.offset = newOffset
	return r.offset, nil
}

func (r *readSeekCloser) Close() error { return nil }
