package amazoncaptcha

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	dirName         = "./captchas"
	failedDir       = "./failed_image"
	trainingDataDir = "./training_data"
	maxWorkers      = 50
)

func initDir(t *testing.T) {
	dirs := []string{
		dirName, failedDir, trainingDataDir,
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.Mkdir(dir, 0777); err != nil {
				t.Fatalf("Failed to create directory %s: %v\n", dir, err)
			}
		}
	}
	// Loop through each letter of the alphabet
	for c := 'A'; c <= 'Z'; c++ {
		// Create the full path to the letter directory
		letterDir := path.Join(trainingDataDir, string(c))

		// Check if the letter directory already exists
		if _, err := os.Stat(letterDir); os.IsNotExist(err) {
			// Create the letter directory if it doesn't exist
			err := os.Mkdir(letterDir, 0777)
			if err != nil {
				fmt.Println(err)
			}
		} else if err != nil {
			// Handle any other errors that occurred
			fmt.Println(err)
		}
	}
}

func TestSolveBatch(t *testing.T) {

	initDir(t)

	files, err := os.ReadDir(dirName)
	if err != nil {
		t.Fatalf("Error reading directory %s: %v\n", dirName, err)
	}

	var testFiles []string
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".jpg") {
			continue
		}
		testFiles = append(testFiles, file.Name())
	}

	var successes, total int32
	var failedFiles []string

	t.Logf("Processing %d files with %d workers...\n", len(testFiles), maxWorkers)

	fileChan := make(chan string, len(testFiles))
	for _, file := range testFiles {
		fileChan <- file
	}
	close(fileChan)

	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				imagePath := filepath.Join(dirName, file)
				imageFile, err := os.ReadFile(imagePath)
				if err != nil {
					t.Logf("Failed to open image file %s: %v", imagePath, err)
					continue
				}
				result, err := Solve(bytes.NewReader(imageFile))
				if err != nil {
					t.Logf("Failed to solve captcha in image file %s: %v", imagePath, err)
					continue
				}
				atomic.AddInt32(&total, 1)
				if result == file[:len(file)-4] {
					atomic.AddInt32(&successes, 1)
				} else {
					failedFiles = append(failedFiles, fmt.Sprintf("%s -> %s", path.Join(dirName, file), result))
					err = os.Rename(imagePath, filepath.Join(failedDir, file))
					if err != nil {
						t.Logf("Failed to move failed image file %s: %v", imagePath, err)
					}
				}
				curSuccesses := atomic.LoadInt32(&successes)
				curTotal := atomic.LoadInt32(&total)
				successRate := float32(curSuccesses) / float32(curTotal) * 100
				progress := float32(curTotal) / float32(len(testFiles)) * 100
				t.Logf("Processing file %s... success rate: %.2f%%, progress: %.2f%%", file, successRate, progress)
			}
		}()
	}

	wg.Wait()

	t.Logf("Processed %d files with success rate: %d/%d (%.2f%%)", total, successes, total, float32(successes)/float32(total)*100)

	if len(failedFiles) > 0 {
		t.Logf("Failed to solve captcha in the following files:")
		for _, file := range failedFiles {
			t.Logf("%s", file)
		}
	}
}

func TestSplitAndSaveCaptchaByLetter(t *testing.T) {

	initDir(t)

	files, err := os.ReadDir(dirName)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	jobs := make(chan string, len(files))

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if !strings.HasSuffix(job, ".jpg") {
					continue
				}
				imgPath := filepath.Join(dirName, job)
				readFile, err := os.ReadFile(imgPath)
				if err != nil {
					t.Logf("Error reading image file %s: %v", imgPath, err)
					continue
				}
				letters, err := FindLetters(bytes.NewReader(readFile))
				if err != nil {
					t.Logf("Failed to split captcha in image file %s: %v", imgPath, err)
					continue
				}
				arr := strings.TrimSuffix(job, ".jpg")
				if len(arr) == 6 && len(letters) > 5 && len(letters) < 8 {
					for x, v := range arr {
						letterPath := filepath.Join(trainingDataDir, string(v))
						filename := uuid.New().String() + ".png"
						filename = filepath.Join(letterPath, filename)
						file, err := os.Create(filename)
						if err != nil {
							t.Logf("Failed to save letter %s for captcha %s: %v", string(v), imgPath, err)
							continue
						}
						defer file.Close()
						err = png.Encode(file, letters[x])
						if err != nil {
							t.Logf("Failed to save letter %s for captcha %s: %v", string(v), imgPath, err)
							continue
						}
					}
				} else {
					_ = os.Remove(imgPath)
					t.Logf("Invalid captcha file name: %s", job)
				}
			}
		}()
	}

	for _, file := range files {
		jobs <- file.Name()
	}

	close(jobs)
	wg.Wait()

	t.Logf("Successfully split and saved captcha letters from %d files", len(files))
}

func TestExtractFeatures(t *testing.T) {

	initDir(t)

	tmpMap := make(map[string][]string)
	files, err := os.ReadDir(trainingDataDir)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if file.IsDir() && len(file.Name()) == 1 && file.Name()[0] >= 'A' && file.Name()[0] <= 'Z' {
			featureList := make([]string, 0)
			subDirPath := filepath.Join(trainingDataDir, file.Name())
			subDirFiles, err := os.ReadDir(subDirPath)
			if err != nil {
				panic(err)
			}
			var wg sync.WaitGroup
			for _, subFile := range subDirFiles {
				if !subFile.IsDir() && strings.HasSuffix(strings.ToLower(subFile.Name()), ".png") {
					imgPath := filepath.Join(subDirPath, subFile.Name())
					wg.Add(1)
					go func(imgPath string) {
						defer wg.Done()
						imgFile, err := os.Open(imgPath)
						if err != nil {
							t.Errorf("open image imgFile error: %s, %v\n", imgPath, err)
							return
						}
						defer imgFile.Close()
						img, err := png.Decode(imgFile)
						if err != nil {
							t.Errorf("png.Decode error: %s, %v\n", imgPath, err)
							_ = os.Remove(imgPath)
							return
						}
						features, err := ExtractFeatures(img.(*image.Gray))
						if err != nil {
							t.Errorf("ExtractFeatures error: %s, %v\n", imgPath, err)
							return
						}
						featureList = append(featureList, features)
					}(imgPath)
				}
			}
			wg.Wait()
			tmpMap[file.Name()] = removeDuplicates(featureList)
		}
	}

	featureMap2 := make(map[string]string)
	for k, arr := range tmpMap {
		for _, v := range arr {
			featureMap2[v] = k
		}
	}

	jsonBytes, err := json.MarshalIndent(featureMap2, "", "	")
	if err != nil {
		t.Errorf("Failed to marshal feature map to json: %v\n", err)
		return
	}

	jsonPath := "./training_data.json"
	if err := os.WriteFile(jsonPath, jsonBytes, 0644); err != nil {
		t.Errorf("Failed to write feature map json file: %v\n", err)
		return
	}

	t.Logf("Extracted features for %d training images and saved to %s", len(files), jsonPath)
}

func TestDownloadCaptchaImages(t *testing.T) {

	var NotFeatures = make(map[string]string)

	client := resty.New()
	client.SetHeaders(map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3",
		"Referer":         "https://www.amazon.com/errors/validateCaptcha",
		"Accept-Language": "en-US,en;q=0.9",
	})

	var wg sync.WaitGroup
	urls := make(chan string, 5000)

	// Start workers to download images
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range urls {
				resp, err := client.R().Get(url)
				if err != nil {
					fmt.Println(err)
					continue
				}
				letters, err := FindLetters(bytes.NewReader(resp.Body()))
				if err != nil {
					fmt.Println(err)
					continue
				}
				if len(letters) < 6 {
					continue
				}
				for _, v := range letters {
					feature, err := ExtractFeatures(v)
					if err != nil {
						panic(err)
					}
					if _, ok := featureMap[feature]; !ok {
						if _, ok := NotFeatures[feature]; ok {
							continue
						}
						NotFeatures[feature] = ""
						filename := fmt.Sprintf("./captchas1/%s", path.Base(url))
						err = os.WriteFile(filename, resp.Body(), os.ModePerm)
						if err != nil {
							panic(err)
						}
						break
					}
				}
			}
		}()
	}

	// Start workers to get image URLs
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5000/maxWorkers; j++ {
				resp, err := client.R().Get("https://www.amazon.com/errors/validateCaptcha")
				if err != nil {
					fmt.Println(err)
					continue
				}
				doc, err := goquery.NewDocumentFromReader(bytes.NewReader(resp.Body()))
				if err != nil {
					fmt.Println(err)
					continue
				}
				captchaImgURL, exists := doc.Find("div.a-row.a-text-center > img").Attr("src")
				if !exists {
					fmt.Println("failed to find captcha image URL")
					continue
				}
				fmt.Println("enqueue -> ", captchaImgURL)

				select {
				case urls <- captchaImgURL:
					// Send URL to the urls channel
				default:
					// Ignore the URL if the urls channel is closed
				}
				//time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	// Wait for all the goroutines to exit before closing the urls channel
	wg.Wait()
	close(urls)

	// Wait for workers to finish downloading images
	wg.Wait()
}

func removeDuplicates(strSlice []string) []string {
	uniqueMap := make(map[string]bool, len(strSlice))
	for _, str := range strSlice {
		if !uniqueMap[str] {
			uniqueMap[str] = true
		}
	}
	uniqueSlice := make([]string, 0, len(uniqueMap))
	for str := range uniqueMap {
		uniqueSlice = append(uniqueSlice, str)
	}
	return uniqueSlice
}

func TestSolveFromImageFile(t *testing.T) {
	// Test the SolveFromImageFile function
	result, err := SolveFromImageFile(path.Join(dirName, "AABTRE.jpg"))
	assert.NoError(t, err)
	assert.Equal(t, "AABTRE", result)
}

func TestSolveFromURL(t *testing.T) {
	// Test the SolveFromURL function
	result, err := SolveFromURL("https://images-na.ssl-images-amazon.com/captcha/sargzmyv/Captcha_kvvvwatlha.jpg")
	assert.NoError(t, err)
	assert.Equal(t, "MYKYAN", result)
}
