package util

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"
)

// Decompress 解压 tar.gz 保留原始的层级结构和文件修改时间
//
// tarFile 被解压的 .tar.gz文件名
//
// dest 解压到哪个目录, 结尾的 "/" 可有可无, "" 和 "./" 和 "." 都表示解压到当前目录
// Decompress 解压 Docker 镜像包到指定的临时目录
// Decompress 解压 tar 文件到指定的临时目录
func Decompress(tarPath, tmpDir string) error {
    // 打开 tar 文件
    file, err := os.Open(tarPath)
    if err != nil {
        return fmt.Errorf("failed to open file %s: %w", tarPath, err)
    }
    defer file.Close()

    // 创建 tar 读取器
    tr := tar.NewReader(file)

    // 解压每个文件
    for {
        header, err := tr.Next()
        if err == io.EOF {
            break // 结束
        }
        if err != nil {
            return fmt.Errorf("failed to read tar header: %w", err)
        }

        // 计算解压后的文件路径
        targetPath := filepath.Join(tmpDir, header.Name)

        // 创建目录或文件
        switch header.Typeflag {
        case tar.TypeDir:
            // 创建目录
            if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
                return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
            }
        case tar.TypeReg:
            // 创建文件
            outFile, err := os.Create(targetPath)
            if err != nil {
                return fmt.Errorf("failed to create file %s: %w", targetPath, err)
            }
            defer outFile.Close()

            // 将内容写入文件
            if _, err := io.Copy(outFile, tr); err != nil {
                return fmt.Errorf("failed to write file %s: %w", targetPath, err)
            }
        }
    }
    return nil
}

func remodifyTime(name string, modTime time.Time) {
	if name == "" {
		return
	}
	atime := time.Now()
	os.Chtimes(name, atime, modTime)
}

func makeDir(name string) (string, error) {
	if name != "" {
		_, err := os.Stat(name)
		if err != nil {
			err = os.MkdirAll(name, 0755)
			if err != nil {
				return "", fmt.Errorf("can not make directory: %v", err)
			}
			return name, nil
		}
		return "", nil
	}
	return "", fmt.Errorf("can not make no name directory: %v", name)
}

func createFile(name string) (*os.File, error) {
	dir := path.Dir(name)
	if dir != "" {
		_, err := os.Lstat(dir)
		if err != nil {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				return nil, err
			}
		}
	}
	return os.Create(name)
}
