package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type UploadParams struct {
	Uploader  *s3manager.Uploader
	S3Bucket  string
	Suffix    string
	WaitGroup *sync.WaitGroup
}

type Remote struct {
	ID        string
	Url       string
	AccessID  string `mapstructure:"access_id"`
	AccessKey string `mapstructure:"access_key"`
}

type AWSConfig struct {
	S3Bucket string `mapstructure:"s3_bucket"`
	S3Region string `mapstructure:"s3_region"`
}

type Config struct {
	AWSConfig AWSConfig `mapstructure:"aws"`
	Remotes   []Remote  `mapstructure:"repo"`
}

var config Config

func main() {
	var configFile string
	pflag.StringVar(&configFile, "config", "", "The config file (default is $HOME/.repo_archive.toml)")
	pflag.Parse()
	if configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {
		// Otherwise, use $HOME/.repo_archive.toml
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		viper.SetConfigName(".repo_archive")
		viper.AddConfigPath(home)
		viper.SetConfigType("toml")
	}
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	} else {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}

	run()
}

func run() {
	err := viper.Unmarshal(&config)
	if err != nil {
		log.Fatal(err)
	}

	session := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String(config.AWSConfig.S3Region),
		},
	}))

	uploadParams := UploadParams{
		Uploader:  s3manager.NewUploader(session),
		S3Bucket:  config.AWSConfig.S3Bucket,
		Suffix:    dateStr(),
		WaitGroup: new(sync.WaitGroup),
	}

	remotes := config.Remotes
	uploadParams.WaitGroup.Add(len(remotes))
	for _, remote := range remotes {
		go func(remote Remote) {
			if err := cloneAndUpload(remote, uploadParams); err != nil {
				log.Println(err)
			}
		}(remote)
	}
	uploadParams.WaitGroup.Wait()
}

func dateStr() string {
	layout := "20060102"
	return time.Now().Format(layout)
}

func mkS3Path(remoteId, suffix string) string {
	return fmt.Sprintf("data/%s/%s-%s.tgz", remoteId, remoteId, suffix)
}

func cloneAndUpload(remote Remote, uploadParams UploadParams) error {
	defer uploadParams.WaitGroup.Done()

	tmpDir, err := ioutil.TempDir("", "repo_archive_"+remote.ID)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	log.Println("cloning", remote.ID)
	coDir := fmt.Sprintf("%s/%s", tmpDir, remote.ID)
	options := &git.CloneOptions{URL: remote.Url}
	if remote.AccessID != "" {
		options.Auth = &http.BasicAuth{
			Username: remote.AccessID,
			Password: remote.AccessKey,
		}
	}

	_, err = git.PlainClone(coDir,
		false,
		options)
	if err != nil {
		log.Println("error cloning", err, remote.Url)
		return err
	}

	tf := fmt.Sprintf("%s/%s.tgz", tmpDir, remote.ID)
	if err = mktarball(coDir, tf); err != nil {
		return err
	}

	log.Println("tarball created at", tf)
	return uploadSingleS3(tf, uploadParams.S3Bucket, mkS3Path(remote.ID, uploadParams.Suffix), uploadParams.Uploader)
}

func uploadSingleS3(local, bucket, remote string, uploader *s3manager.Uploader) error {
	log.Println("uploading", local)
	f, err := os.Open(local)
	if err != nil {
		return err
	}
	defer f.Close()

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(remote),
		Body:   f,
	})

	if err != nil {
		return err
	}
	log.Println("uploaded to", result.Location)

	return nil
}

func mktarball(input, output string) error {
	info, err := os.Stat(input)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", input)
	}

	of, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer of.Close()

	gw := gzip.NewWriter(of)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Ensure the tarball does not start with the absolute path
	prefixLen := 0
	if strings.HasPrefix(input, "/") {
		prefixLen = len(path.Dir(input)) + 1
	}

	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		hdr := &tar.Header{
			Name:    path[prefixLen:],
			Mode:    int64(info.Mode()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if err := writeFile(path, tw); err != nil {
			return err
		}
		return nil
	}
	return filepath.Walk(input, walkFn)
}

func writeFile(from string, w io.Writer) error {
	r, err := os.Open(from)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, r)
	r.Close()
	return err
}
