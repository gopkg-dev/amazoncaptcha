package amazoncaptcha

import (
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"strings"

	_ "image/jpeg"
	_ "image/png"
)

// MonoWeight Define a constant MonoWeight with a value of 1, representing the threshold used to convert grayscale images to binary images.
const MonoWeight = 1

// MaximumLetterLength Define a constant MaximumLetterLength with a value of 33, representing the maximum width of a single letter.
const MaximumLetterLength = 33

// MinimumLetterLength Define a constant MinimumLetterLength with a value of 14, representing the minimum width of the first letter.
// If the width of the first letter is less than this value, all letters will be replaced with blank letters.
const MinimumLetterLength = 14

// FindLetters attempts to locate the letters in a captcha image and returns a slice of grayscale letter images.
// It takes an io.Reader as input, which should contain a valid captcha image.
// It returns a slice of grayscale letter images and an error if the letter extraction process fails.
func FindLetters(r io.Reader) ([]*image.Gray, error) {

	// Decode the input image
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}

	// Convert the input image to grayscale
	grayImg := Grayscale(img)

	// Convert the grayscale image to monochrome using a threshold value
	grayImg = MonoChrome(grayImg, MonoWeight)

	// Find the letter boxes in the monochrome image
	letterBoxes := FindLetterBoxes(grayImg, MaximumLetterLength)

	// Extract the letters from the monochrome image based on the letter boxes
	letters := make([]*image.Gray, len(letterBoxes))
	for i, box := range letterBoxes {

		// Calculate the width and height of the letter box
		width := box.Max.X - box.Min.X
		height := box.Max.Y - box.Min.Y

		// Create a new grayscale image for the letter
		letterImg := image.NewGray(image.Rect(0, 0, width, height))

		// Copy the pixels from the original grayscale image to the new letter image
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				// Calculate the position of the pixel in the original grayscale image
				origX := box.Min.X + x
				origY := box.Min.Y + y

				// Copy the pixel from the original grayscale image to the new letter image
				letterImg.SetGray(x, y, grayImg.GrayAt(origX, origY))
			}
		}

		// Add the new letter image to the letters slice
		letters[i] = letterImg
	}

	// If the number of letters is not exactly 6 or 7, or the width of the first letter is too small,
	// replace all letters with blank letters
	if (len(letters) == 6 && letters[0].Bounds().Dx() < MinimumLetterLength) || (len(letters) != 6 && len(letters) != 7) {
		blankLetter := image.NewGray(image.Rect(0, 0, 200, 70))
		letters = make([]*image.Gray, 6)
		for i := range letters {
			letters[i] = blankLetter
		}
	}

	// If there are 7 letters, merge the first and last letters together
	if len(letters) == 7 {
		// Merge the first and last letters horizontally
		merged, err := MergeHorizontally(letters[6], letters[0])
		if err != nil {
			return nil, err
		}

		// Replace the last letter with the merged letter
		letters[6] = merged

		// Remove the first letter from the slice
		copy(letters[0:], letters[1:])
		letters[len(letters)-1] = nil
		letters = letters[:len(letters)-1]
	}

	// Warning: Commenting out the following line since it may reduce recognition accuracy
	// Remove white borders from each letter image
	// for i, letter := range letters {
	// letters[i] = CutTheWhite(letter)
	// }

	// Join the recognition results into a single string and return it
	return letters, nil
}

// Solve attempts to solve a captcha image and returns a list of character images.
func Solve(r io.Reader) (string, error) {

	// Call the FindLetters function to extract the letter images from the input image
	letters, err := FindLetters(r)
	if err != nil {
		return "", err
	}

	// Define a slice to hold the recognition results
	result := make([]string, len(letters))

	// Loop over each letter image and extract its features
	for i, letter := range letters {
		features, err := ExtractFeatures(letter)
		if err != nil {
			return "", err
		}
		//if v, ok := trainingDataSyncMap.Load(features); ok {
		//	result[i] = v.(string)
		//} else {
		//	result[i] = "-"
		//}
		if v, ok := featureMap[features]; ok {
			result[i] = v
		} else {
			result[i] = "-"
		}
	}

	// Join the recognition results into a single string and return it
	return strings.Join(result, ""), nil
}

// SolveFromImageFile takes a file path of an image file as input, opens the file,
// and processes the data from the image file using the Solve function.
// It returns the processed result as a string and an error if any error occurs during the process.
func SolveFromImageFile(filepath string) (string, error) {
	// Open the image file
	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	// Use the Solve function to process the data from the image file
	result, err := Solve(file)
	if err != nil {
		return "", fmt.Errorf("failed to solve: %w", err)
	}

	return result, nil
}

// SolveFromURL takes a URL string as input, makes an HTTP request to the given URL,
// and processes the data from the URL using the Solve function.
// It returns the processed result as a string and an error if any error occurs during the process.
func SolveFromURL(url string) (string, error) {
	// Make an HTTP request to the given URL
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check the HTTP response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status code: %d", resp.StatusCode)
	}

	// Use the Solve function to process the data from the URL
	result, err := Solve(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to solve: %w", err)
	}

	return result, nil
}
