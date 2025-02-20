package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"log"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
)

type TranslationJob struct {
	SourceLang string
	TargetLang string
	Content    interface{}
}

// 로깅 설정
func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stdout)
}

func main() {
	// .env 파일 로드
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
		return
	}

	// OpenAI API 키 확인
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY is not set in .env file")
		return
	}

	// 1. 소스 JSON 파일 읽기
	sourceFile := "locales/ko/common.json"
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		fmt.Printf("Error reading source file: %v\n", err)
		return
	}

	// 2. JSON 파싱
	var content map[string]interface{}
	if err := json.Unmarshal(data, &content); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	// 3. 대상 언어 리스트 정의
	targetLanguages := []string{"zh", "vi"}

	// 4. OpenAI 클라이언트 초기화
	client := openai.NewClient(apiKey)

	// 5. 각 언어별로 번역 수행
	for _, lang := range targetLanguages {
		translatedContent := translateContent(client, content, "ko", lang)

		// 6. 번역된 내용을 파일로 저장
		outputDir := filepath.Join("locales", lang)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("Error creating directory for %s: %v\n", lang, err)
			continue
		}

		outputFile := filepath.Join(outputDir, "common.json")
		translatedJSON, err := json.MarshalIndent(translatedContent, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling JSON for %s: %v\n", lang, err)
			continue
		}

		if err := os.WriteFile(outputFile, translatedJSON, 0644); err != nil {
			fmt.Printf("Error writing file for %s: %v\n", lang, err)
			continue
		}

		fmt.Printf("Successfully translated and saved to %s\n", outputFile)
	}
}

func translateContent(client *openai.Client, content interface{}, sourceLang, targetLang string) interface{} {
	try := func() (interface{}, error) {
		log.Printf("데이터 처리 시작")

		if content == nil {
			return nil, fmt.Errorf("입력 데이터가 비어있습니다")
		}

		// 전체 콘텐츠를 JSON 문자열로 변환
		jsonContent, err := json.Marshal(content)
		if err != nil {
			return nil, fmt.Errorf("JSON 변환 중 오류: %v", err)
		}

		// 전체 텍스트 번역 수행
		translatedJSON, err := translateText(client, string(jsonContent), sourceLang, targetLang)
		if err != nil {
			return nil, fmt.Errorf("번역 중 오류: %v", err)
		}

		// 번역된 JSON 파싱
		var result interface{}
		if err := json.Unmarshal([]byte(translatedJSON), &result); err != nil {
			return nil, fmt.Errorf("번역된 JSON 파싱 중 오류: %v", err)
		}

		log.Printf("데이터 처리 완료")
		return result, nil
	}

	if result, err := try(); err != nil {
		log.Printf("프로세스 종료: %v", err)
		return content
	} else {
		log.Printf("프로세스 완료")
		return result
	}
}

func translateText(client *openai.Client, text, sourceLang, targetLang string) (string, error) {
	prompt := fmt.Sprintf(`Translate the following JSON from %s to %s. 
Maintain the exact same JSON structure and keys. Only translate the values.
Return ONLY the translated JSON without any additional text or formatting.

JSON to translate:
%s`, sourceLang, targetLang, text)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.3,
		},
	)

	if err != nil {
		return "", fmt.Errorf("Translation error: %v", err)
	}

	// 응답에서 JSON 부분만 추출
	response := resp.Choices[0].Message.Content
	jsonRegex := regexp.MustCompile(`(?s)\{.*\}|\[.*\]`)
	if matches := jsonRegex.FindString(response); matches != "" {
		response = matches
	}

	// JSON 유효성 검사
	var jsonCheck interface{}
	if err := json.Unmarshal([]byte(response), &jsonCheck); err != nil {
		return "", fmt.Errorf("Invalid JSON in response: %v", err)
	}

	return response, nil
}
