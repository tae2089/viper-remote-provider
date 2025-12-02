package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
	viper_remote_provider "github.com/tae2089/viper-remote-provider"
	"github.com/tae2089/viper-remote-provider/provider/github"
)

func main() {
	// var appName string = ""
	// 1. GitHub Provider 옵션 설정
	// GITHUB_TOKEN 환경 변수가 필요합니다.
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN environment variable is required")
	}

	// 예제용 설정입니다. 실제 사용 시에는 본인의 리포지토리 정보로 변경해주세요.
	option := &github.Option{
		Owner:           "tae2089",     // GitHub Owner (User or Org)
		Repository:      "config",      // Repository Name
		Branch:          "main",        // Branch Name
		Path:            "config.yaml", // Config File Path in Repo
		Token:           token,
		PollingInterval: 10 * time.Second, // 10초마다 변경사항 체크
	}

	// 2. 옵션 등록
	// 이 함수를 호출해야 GitHub Provider가 올바르게 초기화됩니다.
	viper_remote_provider.SetOptions(option)

	// 3. Viper에 Remote Provider 추가
	// endpoint는 "github.com"으로 설정하고, path는 리포지토리 내 파일 경로와 일치시킵니다.
	err := viper.AddRemoteProvider("github", "github.com", "config.yaml")
	if err != nil {
		log.Fatalf("Error adding remote provider: %v", err)
	}

	viper.SetConfigType("yaml") // 설정 파일 형식 지정

	// 4. 초기 설정 읽기
	fmt.Println("Reading remote config...")

	fmt.Println("Successfully read config!")
	fmt.Printf("Initial settings: %v\n", viper.AllSettings())
	viper.GetViper().WatchRemoteConfigOnChannel()
	select {}

}
