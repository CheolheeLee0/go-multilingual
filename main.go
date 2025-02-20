package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"log"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
)

const (
	MAX_CONCURRENT_JOBS = 30 // 최대 동시 실행 고루틴 수
	MAX_RETRIES         = 1  // 최대 재시도 횟수
	RETRY_DELAY         = 1  // 재시도 대기 시간(초)
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

// 언어 코드와 이름 매핑
var languageMap = map[string]string{
	"ar":  "Arabic",     // 아랍어
	"bn":  "Bengali",    // 벵골어
	"cs":  "Czech",      // 체코어
	"da":  "Danish",     // 덴마크어
	"de":  "German",     // 독일어
	"el":  "Greek",      // 그리스어
	"en":  "English",    // 영어
	"es":  "Spanish",    // 스페인어
	"fa":  "Persian",    // 페르시아어
	"fi":  "Finnish",    // 핀란드어
	"fil": "Filipino",   // 필리핀어
	"fr":  "French",     // 프랑스어
	"he":  "Hebrew",     // 히브리어
	"hi":  "Hindi",      // 힌디어
	"hu":  "Hungarian",  // 헝가리어
	"id":  "Indonesian", // 인도네시아어
	"it":  "Italian",    // 이탈리아어
	"ja":  "Japanese",   // 일본어
	"ko":  "Korean",     // 한국어
	"km":  "Khmer",      // 크메르어
	"lo":  "Lao",        // 라오어
	"ms":  "Malay",      // 말레이어
	"my":  "Burmese",    // 미얀마어
	"nl":  "Dutch",      // 네덜란드어
	"no":  "Norwegian",  // 노르웨이어
	"pl":  "Polish",     // 폴란드어
	"pt":  "Portuguese", // 포르투갈어
	"ro":  "Romanian",   // 루마니아어
	"ru":  "Russian",    // 러시아어
	"si":  "Sinhala",    // 싱할라어
	"sk":  "Slovak",     // 슬로바키아어
	"sv":  "Swedish",    // 스웨덴어
	"ta":  "Tamil",      // 타밀어
	"te":  "Telugu",     // 텔루구어
	"th":  "Thai",       // 태국어
	"tr":  "Turkish",    // 터키어
	"uk":  "Ukrainian",  // 우크라이나어
	"ur":  "Urdu",       // 우르두어
	"vi":  "Vietnamese", // 베트남어
	"zh":  "Chinese",    // 중국어
	"af":  "Afrikaans",  // 아프리칸스어
	"am":  "Amharic",    // 암하라어
	"bg":  "Bulgarian",  // 불가리아어
	"ca":  "Catalan",    // 카탈로니아어
	"et":  "Estonian",   // 에스토니아어
	"hr":  "Croatian",   // 크로아티아어
	"is":  "Icelandic",  // 아이슬란드어
	"ka":  "Georgian",   // 조지아어
	"lt":  "Lithuanian", // 리투아니아어
	"lv":  "Latvian",    // 라트비아어
}

// var languageMap = map[string]string{
// 	"af":  "Afrikaans",                   // 아프리칸스어
// 	"agq": "Aghem",                       // 아겜어
// 	"ak":  "Akan",                        // 아칸어
// 	"am":  "Amharic",                     // 암하라어
// 	"ar":  "Arabic",                      // 아랍어
// 	"as":  "Assamese",                    // 아삼어
// 	"asa": "Asu",                         // 아수어
// 	"ast": "Asturian",                    // 아스투리아스어
// 	"az":  "Azerbaijani",                 // 아제르바이잔어
// 	"bas": "Basaa",                       // 바사어
// 	"be":  "Belarusian",                  // 벨라루스어
// 	"bem": "Bemba",                       // 벰바어
// 	"bez": "Bena",                        // 베나어
// 	"bg":  "Bulgarian",                   // 불가리아어
// 	"bm":  "Bambara",                     // 밤바라어
// 	"bn":  "Bengali",                     // 벵골어
// 	"bo":  "Tibetan",                     // 티베트어
// 	"br":  "Breton",                      // 브르타뉴어
// 	"brx": "Bodo",                        // 보도어
// 	"bs":  "Bosnian",                     // 보스니아어
// 	"ca":  "Catalan",                     // 카탈로니아어
// 	"ccp": "Chakma",                      // 차크마어
// 	"ce":  "Chechen",                     // 체첸어
// 	"cgg": "Chiga",                       // 치가어
// 	"chr": "Cherokee",                    // 체로키어
// 	"ckb": "Central Kurdish",             // 중앙 쿠르드어
// 	"cs":  "Czech",                       // 체코어
// 	"cy":  "Welsh",                       // 웨일스어
// 	"da":  "Danish",                      // 덴마크어
// 	"dav": "Taita",                       // 타이타어
// 	"de":  "German",                      // 독일어
// 	"dje": "Zarma",                       // 자르마어
// 	"dsb": "Lower Sorbian",               // 저지 소르브어
// 	"dua": "Duala",                       // 두알라어
// 	"dyo": "Jola-Fonyi",                  // 졸라-포니어
// 	"dz":  "Dzongkha",                    // 종카어
// 	"ebu": "Embu",                        // 엠부어
// 	"ee":  "Ewe",                         // 에웨어
// 	"el":  "Greek",                       // 그리스어
// 	"en":  "English",                     // 영어
// 	"eo":  "Esperanto",                   // 에스페란토어
// 	"es":  "Spanish",                     // 스페인어
// 	"et":  "Estonian",                    // 에스토니아어
// 	"eu":  "Basque",                      // 바스크어
// 	"ewo": "Ewondo",                      // 에원도어
// 	"fa":  "Persian",                     // 페르시아어
// 	"ff":  "Fulah",                       // 풀라어
// 	"fi":  "Finnish",                     // 핀란드어
// 	"fil": "Filipino",                    // 필리핀어
// 	"fo":  "Faroese",                     // 페로어
// 	"fr":  "French",                      // 프랑스어
// 	"fur": "Friulian",                    // 프리울리어
// 	"fy":  "Western Frisian",             // 서프리지아어
// 	"ga":  "Irish",                       // 아일랜드어
// 	"gd":  "Scottish Gaelic",             // 스코틀랜드 게일어
// 	"gl":  "Galician",                    // 갈리시아어
// 	"gsw": "Swiss German",                // 스위스 독일어
// 	"gu":  "Gujarati",                    // 구자라트어
// 	"guz": "Gusii",                       // 구시어
// 	"gv":  "Manx",                        // 맨섬어
// 	"ha":  "Hausa",                       // 하우사어
// 	"haw": "Hawaiian",                    // 하와이어
// 	"he":  "Hebrew",                      // 히브리어
// 	"hi":  "Hindi",                       // 힌디어
// 	"hr":  "Croatian",                    // 크로아티아어
// 	"hsb": "Upper Sorbian",               // 고지 소르브어
// 	"hu":  "Hungarian",                   // 헝가리어
// 	"hy":  "Armenian",                    // 아르메니아어
// 	"id":  "Indonesian",                  // 인도네시아어
// 	"ig":  "Igbo",                        // 이그보어
// 	"ii":  "Sichuan Yi",                  // 쓰촨 이어
// 	"is":  "Icelandic",                   // 아이슬란드어
// 	"it":  "Italian",                     // 이탈리아어
// 	"ja":  "Japanese",                    // 일본어
// 	"jgo": "Ngomba",                      // 응곰바어
// 	"jmc": "Machame",                     // 마차메어
// 	"ka":  "Georgian",                    // 조지아어
// 	"kab": "Kabyle",                      // 카빌어
// 	"kam": "Kamba",                       // 캄바어
// 	"kde": "Makonde",                     // 마콘데어
// 	"kea": "Kabuverdianu",                // 카보베르데어
// 	"khq": "Koyra Chiini",                // 코이라 치니어
// 	"ki":  "Kikuyu",                      // 키쿠유어
// 	"kk":  "Kazakh",                      // 카자흐어
// 	"kkj": "Kako",                        // 카코어
// 	"kl":  "Kalaallisut",                 // 그린란드어
// 	"kln": "Kalenjin",                    // 칼렌진어
// 	"km":  "Khmer",                       // 크메르어
// 	"kn":  "Kannada",                     // 칸나다어
// 	"ko":  "Korean",                      // 한국어
// 	"kok": "Konkani",                     // 콘칸어
// 	"ks":  "Kashmiri",                    // 카슈미르어
// 	"ksb": "Shambala",                    // 샴발라어
// 	"ksf": "Bafia",                       // 바피아어
// 	"ksh": "Colognian",                   // 쾰른어
// 	"kw":  "Cornish",                     // 콘월어
// 	"ky":  "Kyrgyz",                      // 키르기스어
// 	"lag": "Langi",                       // 랑기어
// 	"lb":  "Luxembourgish",               // 룩셈부르크어
// 	"lg":  "Ganda",                       // 간다어
// 	"lkt": "Lakota",                      // 라코타어
// 	"ln":  "Lingala",                     // 링갈라어
// 	"lo":  "Lao",                         // 라오어
// 	"lrc": "Northern Luri",               // 북부 루리어
// 	"lt":  "Lithuanian",                  // 리투아니아어
// 	"lu":  "Luba-Katanga",                // 루바-카탕가어
// 	"luo": "Luo",                         // 루오어
// 	"luy": "Luyia",                       // 루이아어
// 	"lv":  "Latvian",                     // 라트비아어
// 	"mas": "Masai",                       // 마사이어
// 	"mer": "Meru",                        // 메루어
// 	"mfe": "Morisyen",                    // 모리셔스 크레올어
// 	"mg":  "Malagasy",                    // 말라가시어
// 	"mgh": "Makhuwa-Meetto",              // 마쿠아-메토어
// 	"mgo": "Metaʼ",                       // 메타어
// 	"mk":  "Macedonian",                  // 마케도니아어
// 	"ml":  "Malayalam",                   // 말라얄람어
// 	"mn":  "Mongolian",                   // 몽골어
// 	"mr":  "Marathi",                     // 마라티어
// 	"ms":  "Malay",                       // 말레이어
// 	"mt":  "Maltese",                     // 몰타어
// 	"mua": "Mundang",                     // 문당어
// 	"my":  "Burmese",                     // 미얀마어
// 	"mzn": "Mazanderani",                 // 마잔데라니어
// 	"naq": "Nama",                        // 나마어
// 	"nb":  "Norwegian Bokmål",            // 노르웨이 부크몰
// 	"nd":  "North Ndebele",               // 북부 은데벨레어
// 	"nds": "Low German",                  // 저지 독일어
// 	"ne":  "Nepali",                      // 네팔어
// 	"nl":  "Dutch",                       // 네덜란드어
// 	"nmg": "Kwasio",                      // 콰시오어
// 	"nn":  "Norwegian Nynorsk",           // 노르웨이 뉘노르스크
// 	"nnh": "Ngiemboon",                   // 느기엠분어
// 	"nus": "Nuer",                        // 누에르어
// 	"nyn": "Nyankole",                    // 냔콜레어
// 	"om":  "Oromo",                       // 오로모어
// 	"or":  "Odia",                        // 오디아어
// 	"os":  "Ossetic",                     // 오세트어
// 	"pa":  "Punjabi",                     // 펀자브어
// 	"pl":  "Polish",                      // 폴란드어
// 	"ps":  "Pashto",                      // 파슈토어
// 	"pt":  "Portuguese",                  // 포르투갈어
// 	"qu":  "Quechua",                     // 케추아어
// 	"rm":  "Romansh",                     // 로만시어
// 	"rn":  "Rundi",                       // 룬디어
// 	"ro":  "Romanian",                    // 루마니아어
// 	"rof": "Rombo",                       // 롬보어
// 	"ru":  "Russian",                     // 러시아어
// 	"rw":  "Kinyarwanda",                 // 키냐르완다어
// 	"rwk": "Rwa",                         // 르와어
// 	"sah": "Sakha",                       // 사하어
// 	"saq": "Samburu",                     // 삼부루어
// 	"sbp": "Sangu",                       // 상구어
// 	"se":  "Northern Sami",               // 북부 사미어
// 	"seh": "Sena",                        // 세나어
// 	"ses": "Koyraboro Senni",             // 코이라보로 세니어
// 	"sg":  "Sango",                       // 상고어
// 	"shi": "Tachelhit",                   // 타셸히트어
// 	"si":  "Sinhala",                     // 싱할라어
// 	"sk":  "Slovak",                      // 슬로바키아어
// 	"sl":  "Slovenian",                   // 슬로베니아어
// 	"smn": "Inari Sami",                  // 이나리 사미어
// 	"sn":  "Shona",                       // 쇼나어
// 	"so":  "Somali",                      // 소말리어
// 	"sq":  "Albanian",                    // 알바니아어
// 	"sr":  "Serbian",                     // 세르비아어
// 	"sv":  "Swedish",                     // 스웨덴어
// 	"sw":  "Swahili",                     // 스와힐리어
// 	"ta":  "Tamil",                       // 타밀어
// 	"te":  "Telugu",                      // 텔루구어
// 	"teo": "Teso",                        // 테소어
// 	"tg":  "Tajik",                       // 타지크어
// 	"th":  "Thai",                        // 태국어
// 	"ti":  "Tigrinya",                    // 티그리냐어
// 	"to":  "Tongan",                      // 통가어
// 	"tr":  "Turkish",                     // 터키어
// 	"tt":  "Tatar",                       // 타타르어
// 	"twq": "Tasawaq",                     // 타사와크어
// 	"tzm": "Central Atlas Tamazight",     // 중앙 아틀라스 타마지트어
// 	"ug":  "Uyghur",                      // 위구르어
// 	"uk":  "Ukrainian",                   // 우크라이나어
// 	"ur":  "Urdu",                        // 우르두어
// 	"uz":  "Uzbek",                       // 우즈베크어
// 	"vai": "Vai",                         // 바이어
// 	"vi":  "Vietnamese",                  // 베트남어
// 	"vun": "Vunjo",                       // 분조어
// 	"wae": "Walser",                      // 발저어
// 	"wo":  "Wolof",                       // 월로프어
// 	"xog": "Soga",                        // 소가어
// 	"yav": "Yangben",                     // 양벤어
// 	"yi":  "Yiddish",                     // 이디시어
// 	"yo":  "Yoruba",                      // 요루바어
// 	"yue": "Cantonese",                   // 광둥어
// 	"zgh": "Standard Moroccan Tamazight", // 표준 모로코 타마지트어
// 	"zh":  "Chinese",                     // 중국어
// 	"zu":  "Zulu",                        // 줄루어
// }

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
	sourceFile := "locales/en/common.json"
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
	//    // 3. 대상 언어 리스트 정의
	// var targetLanguages []string
	// for langCode := range languageMap {
	// 	if langCode != "en" { // 소스 언어(한국어)는 제외
	// 		targetLanguages = append(targetLanguages, langCode)
	// 	}
	// }
	// 실패한 언어 목록
	targetLanguages := []string{
		"id", "vi", "ur", "km", "my", "fil", "el", "ms",
		"pl", "si",
	}

	// targetLanguages := []string{
	// 	"rm", "kw", "ml",
	// }

	// 4. OpenAI 클라이언트 초기화
	client := openai.NewClient(apiKey)

	// 진행 상황 추적을 위한 변수들
	totalLanguages := len(targetLanguages)
	completedCount := 0
	successCount := 0
	var progressMutex sync.Mutex

	// 번역 결과와 에러를 저장할 채널 생성
	type translationResult struct {
		lang    string
		content interface{}
		err     error
	}
	resultChan := make(chan translationResult, len(targetLanguages))

	// 세마포어 생성
	sem := make(chan struct{}, MAX_CONCURRENT_JOBS)
	var wg sync.WaitGroup

	// 진행 상황 출력 함수
	printProgress := func(lang string, success bool, err error) {
		completedCount++
		if success {
			successCount++
		}

		percentage := float64(completedCount) / float64(totalLanguages) * 100
		fmt.Printf("\n=== 번역 진행 상황 ===\n")
		fmt.Printf("총 언어: %d개\n", totalLanguages)
		fmt.Printf("완료된 언어: %d개 (%.1f%%)\n", completedCount, percentage)
		fmt.Printf("성공: %d개, 실패: %d개\n", successCount, completedCount-successCount)
		fmt.Printf("현재 처리 중인 언어: %s (%s)\n", languageMap[lang], lang)
		if err != nil {
			fmt.Printf("오류 내용: %v\n", err)
		}
		fmt.Printf("=====================\n\n")
	}

	// 5. 각 언어별로 고루틴을 사용하여 동시 번역 수행
	for _, lang := range targetLanguages {
		sem <- struct{}{} // 세마포어 획득
		wg.Add(1)
		go func(lang string) {
			defer wg.Done()
			defer func() { <-sem }() // 세마포어 반환

			var translatedContent interface{}
			var err error

			// 재시도 로직
			for retry := 0; retry < MAX_RETRIES; retry++ {
				translatedContent, err = translateContent(client, content, "en", lang)
				if err == nil {
					break
				}

				log.Printf("Translation retry %d for language %s: %v", retry+1, lang, err)
				if retry < MAX_RETRIES-1 {
					time.Sleep(time.Second * RETRY_DELAY)
				}
			}

			// 진행 상황 출력을 위한 뮤텍스 잠금
			progressMutex.Lock()
			printProgress(lang, err == nil, err)
			progressMutex.Unlock()

			// 결과 전송
			resultChan <- translationResult{lang: lang, content: translatedContent, err: err}
		}(lang)
	}

	// 모든 고루틴이 완료될 때까지 대기
	wg.Wait()
	close(resultChan)

	// 번역 실패한 언어를 저장할 슬라이스
	var failedLanguages []string

	// 모든 번역 결과 수집
	for result := range resultChan {
		if result.err != nil {
			fmt.Printf("Translation failed for language %s (%s): %v\n", languageMap[result.lang], result.lang, result.err)
			failedLanguages = append(failedLanguages, result.lang)
			continue
		}

		// 6. 번역된 내용을 파일로 저장
		outputDir := filepath.Join("locales", result.lang)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("Error creating directory for %s: %v\n", result.lang, err)
			failedLanguages = append(failedLanguages, result.lang)
			continue
		}

		outputFile := filepath.Join(outputDir, "common.json")
		translatedJSON, err := json.MarshalIndent(result.content, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling JSON for %s: %v\n", result.lang, err)
			continue
		}

		// 파일 존재 여부 확인
		if _, err := os.Stat(outputFile); err == nil {
			fmt.Printf("Overwriting existing file: %s\n", outputFile)
		}

		if err := os.WriteFile(outputFile, translatedJSON, 0644); err != nil {
			fmt.Printf("Error writing file for %s: %v\n", result.lang, err)
			failedLanguages = append(failedLanguages, result.lang)
			continue
		}

		fmt.Printf("Successfully translated and saved to %s\n", outputFile)
	}

	// 번역 실패한 언어가 있다면 출력
	if len(failedLanguages) > 0 {
		fmt.Println("\nTranslation failed for the following languages:")
		for _, lang := range failedLanguages {
			fmt.Printf("- %s (%s)\n", languageMap[lang], lang)
		}
	}
}

func translateContent(client *openai.Client, content interface{}, sourceLang, targetLang string) (interface{}, error) {
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

	result, err := try()
	if err != nil {
		log.Printf("프로세스 종료: %v", err)
		return nil, err
	}
	log.Printf("프로세스 완료")
	return result, nil
}

func translateText(client *openai.Client, text, sourceLang, targetLang string) (string, error) {
	sourceLanguage := languageMap[sourceLang]
	targetLanguage := languageMap[targetLang]

	if sourceLanguage == "" {
		sourceLanguage = sourceLang
	}
	if targetLanguage == "" {
		targetLanguage = targetLang
	}

	prompt := fmt.Sprintf(`You are a professional translator specializing in B2B SaaS localization.

Task: Translate the following JSON from %s (%s) to %s (%s) while maintaining the following requirements:

Brand Voice Guidelines:
- Professional yet approachable tone
- Clear and concise language
- Maintain technical accuracy for B2B SaaS context
- Keep marketing messages persuasive and solution-focused
- Preserve formal business language while being engaging

Translation Requirements:
1. Maintain exact JSON structure and keys (do not translate keys)
2. Only translate the values
3. Preserve any placeholders like {language}, {number}, {step}
4. Keep HTML tags and formatting intact
5. Maintain line breaks indicated by \n
6. Keep technical terms consistent throughout
7. Adapt cultural nuances appropriately for the target language
8. Preserve any numerical values and units

IMPORTANT: Return ONLY the raw JSON without any markdown formatting or code blocks.
Do not wrap the response in `+"```json```"+` tags.

Source JSON to translate:
%s`, sourceLanguage, sourceLang, targetLanguage, targetLang, text)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
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

	log.Printf("응답: %s", resp.Choices[0].Message.Content)

	response := resp.Choices[0].Message.Content

	// 백틱으로 둘러싸인 코드 블록 제거
	response = regexp.MustCompile("```(?:json)?\n?|\n?```").ReplaceAllString(response, "")

	// 응답 트리밍
	response = strings.TrimSpace(response)

	// JSON 유효성 검사
	if !json.Valid([]byte(response)) {
		return "", fmt.Errorf("Invalid JSON structure in response: %s", response)
	}

	// JSON 포맷팅
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(response), "", "  "); err != nil {
		return "", fmt.Errorf("Error formatting JSON: %v", err)
	}

	return prettyJSON.String(), nil
}
