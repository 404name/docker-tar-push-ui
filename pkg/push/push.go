package push

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"image-upload-portal/pkg/util"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/mholt/archiver/v3"
	"github.com/olahol/melody"
	"github.com/opencontainers/go-digest"
	"github.com/silenceper/log"
)

type ImagePush struct {
	archivePath      string
	registryEndpoint string
	username         string
	password         string
	skipSSLVerify    bool
	tmpDir           string
	httpClient       *http.Client
	imagePrefix      string // 指定镜像仓库名称
	session          *melody.Session
	authToken        string
}

// NewImagePush new
func NewImagePush(archivePath, registryEndpoint, imagePrefix, username, password string, skipSSLVerify bool, s *melody.Session) *ImagePush {
	registryEndpoint = strings.TrimSuffix(registryEndpoint, "/")
	// registryEndpoint 如果没有协议，则默认添加https://
	if !strings.HasPrefix(registryEndpoint, "http://") && !strings.HasPrefix(registryEndpoint, "https://") {
		registryEndpoint = "https://" + registryEndpoint
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSSLVerify},
	}
	return &ImagePush{
		archivePath:      archivePath,
		registryEndpoint: registryEndpoint,
		username:         username,
		password:         password,
		skipSSLVerify:    skipSSLVerify,
		tmpDir:           "./tmp/",
		httpClient:       &http.Client{Transport: tr},
		imagePrefix:      imagePrefix,
		// 用于跟前端实时推送日志的ws session
		session: s,
	}
}

// Manifest manifest.json
type Manifest struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

// 用于跟前端实时推送日志的log实现
func (imagePush *ImagePush) Errorf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	if imagePush.session != nil && !imagePush.session.IsClosed() {
		imagePush.session.Write([]byte("[ERROR] " + fmt.Sprintf(format, v...) + "\n"))
	}
}

func (imagePush *ImagePush) Infof(format string, v ...interface{}) {
	log.Infof(format, v...)
	if imagePush.session != nil && !imagePush.session.IsClosed() {
		imagePush.session.Write([]byte("[INFO] " + fmt.Sprintf(format, v...) + "\n"))
	}
}

func (imagePush *ImagePush) Debugf(format string, v ...interface{}) {
	log.Debugf(format, v...)
	// Debug暂时不打印到界面
	// if imagePush.session != nil && !imagePush.session.IsClosed() {
	// 	imagePush.session.Write([]byte("[DEBUG] " + fmt.Sprintf(format, v...) + "\n"))
	// }
}

// Push push archive image
func (imagePush *ImagePush) Push() {
	//判断tar包是否正常
	if !util.Exists(imagePush.archivePath) {
		imagePush.Errorf("%s not exists", imagePush.archivePath)
		return
	}
	imageFiles, err := util.FilesPath(imagePush.archivePath)
	if err != nil {
		imagePush.Errorf("get image FilesPath err: %v", err)
		return
	}
	for _, imagepath := range imageFiles {
		imagePush.preHandle(imagepath)
	}
}

// push预先处理
func (imagePush *ImagePush) preHandle(imagepath string) error {
	imagePush.tmpDir = fmt.Sprintf("./tmp/docker-tar-push/%d", time.Now().UnixNano())
	imagePush.Infof("extract archive file %s to %s", imagepath, imagePush.tmpDir)

	defer func() {
		err := os.RemoveAll(imagePush.tmpDir)
		if err != nil {
			imagePush.Errorf("remove tmp dir %s error, %v", imagePush.tmpDir, err)
		}
	}()

	err := archiver.Unarchive(imagepath, imagePush.tmpDir)
	if err != nil {
		// imagePush.Errorf("tar unarchive failed, %+v", err)
		imagePush.Infof("use gzip unarchive file %s to %s", imagepath, imagePush.tmpDir)
	}
	// gzip 解压
	if err != nil {
		if err = util.Decompress(imagepath, imagePush.tmpDir); err != nil {
			imagePush.Errorf("gzip unarchive failed, %+v", err)
			return err
		}
	}

	// 封装image
	data, err := ioutil.ReadFile(path.Join(imagePush.tmpDir, "manifest.json"))
	if err != nil {
		imagePush.Errorf("read manifest.json failed, %+v", err)
		return err
	}
	var manifestObjs []*Manifest
	err = json.Unmarshal(data, &manifestObjs)
	if err != nil {
		imagePush.Errorf("unmarshal manifest.json failed, %+v", err)
		return err
	}

	for _, manifestObj := range manifestObjs {
		imagePush.Infof("start push image archive %s", imagePush.archivePath)
		for _, repo := range manifestObj.RepoTags {
			//repo = "xxxxxx/test-tar:test-tag"
			image, tag := util.ParseImageAndTag(repo)
			// image := "xxxxx/xxxxxxx/test-tar" 只保留最后部分test-tar
			image = path.Base(image)
			repoImage := path.Join(imagePush.imagePrefix, image)
			imagePush.Debugf("image=%s,tag=%s", image, tag)

			//push layer
			var layerPaths []string
			for _, layer := range manifestObj.Layers {
				if err := imagePush.checkTaskProgress(); err != nil {
					return err
				}
				layerPath := path.Join(imagePush.tmpDir, layer)
				err = imagePush.pushLayer(layer, repoImage)
				if err != nil {
					imagePush.Errorf("pushLayer %s Failed, %v", layer, err)
					return err
				}
				layerPaths = append(layerPaths, layerPath)
			}
			if err := imagePush.checkTaskProgress(); err != nil {
				return err
			}
			//push image config
			err = imagePush.pushConfig(manifestObj.Config, repoImage)
			if err != nil {
				imagePush.Errorf("push image config failed,%+v", err)
				return err
			}
			if err := imagePush.checkTaskProgress(); err != nil {
				return err
			}
			//push manifest
			imagePush.Infof("start push manifest")
			err = imagePush.pushManifest(layerPaths, manifestObj.Config, repoImage, tag)
			if err != nil {
				imagePush.Errorf("push manifest error,%+v", err)
			}
			imagePush.Infof("push manifest done")
		}
	}
	imagePush.Infof("push image archive %s done\n\n", imagepath)
	return nil
}

// 检查当前任务是否正在运行
func (imagePush *ImagePush) checkTaskProgress() error {
	if imagePush.session != nil {
		if inProgress, exists := imagePush.session.Get("taskInProgress"); exists && !inProgress.(bool) {
			return fmt.Errorf("任务中止，镜像停止上传")
		}
	}
	return nil
}

func (imagePush *ImagePush) checkLayerExist(file, image string) (bool, error) {
	hash, err := util.Sha256Hash(file)
	if err != nil {
		log.Errorf("Failed to calculate SHA256 hash for file %s: %v", file, err)
		return false, err
	}

	// 构造 URL
	url := fmt.Sprintf("%s/v2/%s/blobs/%s", imagePush.registryEndpoint, image, fmt.Sprintf("sha256:%s", hash))
	log.Infof("Constructed URL: %s", url)

	// 创建 HTTP 请求
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		log.Errorf("Failed to create HTTP request: %v", err)
		return false, err
	}

	// 设置 Basic Auth
	req.SetBasicAuth(imagePush.username, imagePush.password)
	if imagePush.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+imagePush.authToken)
	}
	log.Infof("Set Basic Auth with username: %s", imagePush.username)

	// 发送请求
	log.Infof("Sending request to %s", url)
	resp, err := imagePush.httpClient.Do(req)
	if err != nil {
		log.Errorf("Failed to send request: %v", err)
		return false, err
	}
	defer resp.Body.Close()

	log.Infof("Received response with status code: %d", resp.StatusCode)

	// 只第一次的时候处理 401 认证失败，通过失败的authHeader拿到认证地址获取到token后保存在 imagePush.authToken 中后续使用
	if resp.StatusCode == http.StatusUnauthorized {
		authHeader := resp.Header.Get("Www-Authenticate")
		if authHeader != "" {
			log.Infof("Received Www-Authenticate header: %s", authHeader)
			token, err := getToken(authHeader, imagePush.username, imagePush.password)
			if err != nil {
				log.Errorf("Failed to get token: %v", err)
				return false, err
			}
			imagePush.authToken = token
			log.Infof("Successfully obtained token: %s", token)
			// 使用 Token 重新发送请求
			req.Header.Set("Authorization", "Bearer "+token)
			log.Infof("Sending request with Bearer token...")
			resp, err = imagePush.httpClient.Do(req)
			if err != nil {
				log.Errorf("Failed to send request with Bearer token: %v", err)
				return false, err
			}
			defer resp.Body.Close()

			log.Infof("Received response with status code: %d", resp.StatusCode)
		}
	}

	// 处理响应
	switch resp.StatusCode {
	case http.StatusOK:
		log.Infof("Layer exists: %s", url)
		return true, nil
	case http.StatusNotFound:
		log.Infof("Layer does not exist: %s", url)
		return false, nil
	default:
		log.Errorf("Unexpected status code: %d for URL: %s", resp.StatusCode, url)
		return false, fmt.Errorf("head %s failed, statusCode is %d", url, resp.StatusCode)
	}
}

func (imagePush *ImagePush) pushManifest(layersPaths []string, imageConfig, image, tag string) error {
	configPath := path.Join(imagePush.tmpDir, imageConfig)
	obj := &schema2.Manifest{}
	obj.SchemaVersion = schema2.SchemaVersion.SchemaVersion
	obj.MediaType = schema2.MediaTypeManifest
	obj.Config.MediaType = schema2.MediaTypeImageConfig
	configSize, err := util.GetFileSize(configPath)
	if err != nil {
		return err
	}
	obj.Config.Size = configSize
	hash, err := util.Sha256Hash(configPath)
	if err != nil {
		return err
	}
	obj.Config.Digest = digest.Digest("sha256:" + hash)
	for _, layersPath := range layersPaths {
		layerSize, err := util.GetFileSize(layersPath)
		if err != nil {
			return err
		}
		hash, err := util.Sha256Hash(layersPath)
		if err != nil {
			return err
		}
		item := distribution.Descriptor{
			MediaType: schema2.MediaTypeUncompressedLayer,
			Size:      layerSize,
			Digest:    digest.Digest("sha256:" + hash),
		}
		obj.Layers = append(obj.Layers, item)
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", imagePush.registryEndpoint, image, tag)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		imagePush.Errorf("push manifest request error,%+v", err)
		return err
	}
	req.SetBasicAuth(imagePush.username, imagePush.password)
	if imagePush.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+imagePush.authToken)
	}
	imagePush.Debugf("PUT %s", url)
	req.Header.Set("Content-Type", schema2.MediaTypeManifest)
	resp, err := imagePush.httpClient.Do(req)
	if err != nil {
		imagePush.Errorf("push manifest post error,%+v", err)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		imagePush.Errorf("read url /manifests/ body err: %v", err)
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("put manifest failed, code is %d, body: %v", resp.StatusCode, string(body))
	}
	return nil
}

func (imagePush *ImagePush) pushConfig(imageConfig, image string) error {
	configPath := path.Join(imagePush.tmpDir, imageConfig)
	// check image config exists
	exist, err := imagePush.checkLayerExist(configPath, image)
	if err != nil {
		return fmt.Errorf("check layer exist failed,%+v", err)

	}
	if exist {
		imagePush.Infof("%s Already exist", imageConfig)
		return nil
	}

	imagePush.Infof("start push image config %s", imageConfig)
	url, err := imagePush.startPushing(image)
	if err != nil {
		return fmt.Errorf("startPushing Error, %+v", err)
	}
	return imagePush.chunkUpload(configPath, url)
}

func (imagePush *ImagePush) pushLayer(layer, image string) error {
	layerPath := path.Join(imagePush.tmpDir, layer)
	// check layer exists
	exist, err := imagePush.checkLayerExist(layerPath, image)
	if err != nil {
		return fmt.Errorf("check layer exist failed,%+v", err)
	}
	if exist {
		imagePush.Infof("%s Already exist", layer)
		return nil
	}
	url, err := imagePush.startPushing(image)
	if err != nil {
		return fmt.Errorf("startPushing Error, %+v", err)
	}
	return imagePush.chunkUpload(layerPath, url)
}

func (imagePush *ImagePush) chunkUpload(file, url string) error {
	imagePush.Debugf("push file %s to %s", file, url)
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	stat, err := f.Stat() //获取文件状态
	if err != nil {
		return err
	}
	defer f.Close()
	contentSize := stat.Size()
	chunkSize := 2097152
	index, offset := 0, 0
	buf := make([]byte, chunkSize)
	h := sha256.New()
	for {
		n, err := f.Read(buf)
		if err == io.EOF {
			break
		}
		offset = index + n
		index = offset
		imagePush.Infof("Pushing %s ... %.2f%s", file, (float64(offset)/float64(contentSize))*100, "%")

		chunk := buf[0:n]

		h.Write(chunk)

		if int64(offset) == contentSize {
			sum := h.Sum(nil)
			//由于是十六进制表示，因此需要转换
			hash := hex.EncodeToString(sum)
			//last
			req, err := http.NewRequest("PUT",
				fmt.Sprintf("%s&digest=sha256:%s", url, hash), bytes.NewBuffer(chunk))
			if err != nil {
				return err
			}
			req.SetBasicAuth(imagePush.username, imagePush.password)
			if imagePush.authToken != "" {
				req.Header.Set("Authorization", "Bearer "+imagePush.authToken)
			}
			imagePush.Debugf("PUT %s", url)
			req.Header.Set("Content-Type", "application/octet-stream")
			req.Header.Set("Content-Length", fmt.Sprintf("%d", n))
			req.Header.Set("Content-Range", fmt.Sprintf("%d-%d", index, offset))
			resp, err := imagePush.httpClient.Do(req)
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusCreated {
				return fmt.Errorf("PUT chunk layer error,code is %d", resp.StatusCode)
			}
			break
		} else {
			req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(chunk))
			if err != nil {
				return err
			}
			req.SetBasicAuth(imagePush.username, imagePush.password)
			if imagePush.authToken != "" {
				req.Header.Set("Authorization", "Bearer "+imagePush.authToken)
			}
			req.Header.Set("Content-Type", "application/octet-stream")
			req.Header.Set("Content-Length", fmt.Sprintf("%d", n))
			req.Header.Set("Content-Range", fmt.Sprintf("%d-%d", index, offset))
			imagePush.Debugf("PATCH %s", url)
			resp, err := imagePush.httpClient.Do(req)
			if err != nil {
				return err
			}
			location := resp.Header.Get("Location")
			if resp.StatusCode == http.StatusAccepted && location != "" {
				url = location
			} else {
				return fmt.Errorf("PATCH chunk file error,code is %d", resp.StatusCode)
			}
		}
	}
	return nil
}

// 这里格外再判断一次401，防止前面的认证失败，代码后续可以优化
func (imagePush *ImagePush) startPushing(image string) (string, error) {
	// 构造 URL
	url := fmt.Sprintf("%s/v2/%s/blobs/uploads/", imagePush.registryEndpoint, image)
	log.Infof("Constructed upload URL: %s", url)

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Errorf("Failed to create upload request: %v", err)
		return "", fmt.Errorf("failed to create upload request: %v", err)
	}

	// 设置认证信息
	if imagePush.authToken != "" {
		// 使用 Bearer Token 认证
		req.Header.Set("Authorization", "Bearer "+imagePush.authToken)
		log.Infof("Using Bearer token for authentication")
	} else {
		// 使用 Basic Auth 认证
		req.SetBasicAuth(imagePush.username, imagePush.password)
		log.Infof("Using Basic Auth for authentication with username: %s", imagePush.username)
	}

	// 发送请求
	log.Infof("Sending upload request to %s", url)
	resp, err := imagePush.httpClient.Do(req)
	if err != nil {
		log.Errorf("Failed to send upload request: %v", err)
		return "", fmt.Errorf("failed to send upload request: %v", err)
	}
	defer resp.Body.Close()

	log.Infof("Received upload response with status code: %d", resp.StatusCode)

	// 处理 401 认证失败
	if resp.StatusCode == http.StatusUnauthorized {
		log.Infof("Authentication failed, attempting to get new token...")

		// 获取新的 Token
		authHeader := resp.Header.Get("Www-Authenticate")
		if authHeader != "" {
			token, err := getToken(authHeader, imagePush.username, imagePush.password)
			if err != nil {
				log.Errorf("Failed to get new token: %v", err)
				return "", fmt.Errorf("failed to get new token: %v", err)
			}

			// 更新 authToken
			imagePush.authToken = token
			log.Infof("Successfully obtained new token: %s", token)

			// 使用新的 Token 重新发送请求
			req.Header.Set("Authorization", "Bearer "+token)
			log.Infof("Retrying upload request with new token...")
			resp, err = imagePush.httpClient.Do(req)
			if err != nil {
				log.Errorf("Failed to retry upload request: %v", err)
				return "", fmt.Errorf("failed to retry upload request: %v", err)
			}
			defer resp.Body.Close()

			log.Infof("Received retry upload response with status code: %d", resp.StatusCode)
		} else {
			log.Errorf("Www-Authenticate header is missing in response")
			return "", fmt.Errorf("Www-Authenticate header is missing in response")
		}
	}

	// 检查响应状态码
	if resp.StatusCode == http.StatusAccepted {
		// 获取 Location 头
		location := resp.Header.Get("Location")
		if location != "" {
			log.Infof("Upload initiated successfully, location: %s", location)
			return location, nil
		} else {
			log.Errorf("Location header is missing in response")
			return "", fmt.Errorf("location header is missing in response")
		}
	} else {
		// 读取响应体以获取更多错误信息
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Failed to read response body: %v", err)
			return "", fmt.Errorf("failed to read response body: %v", err)
		}

		log.Errorf("Upload failed with status code: %d, response body: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("upload failed with status code: %d, response body: %s", resp.StatusCode, string(body))
	}
}
