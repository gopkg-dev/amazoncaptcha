package amazoncaptcha

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
)

// Grayscale generates a grayscale version of an image.
func Grayscale(img image.Image) *image.Gray {
	// Create a new grayscale image with the same bounds as the input image
	grayImg := image.NewGray(img.Bounds())

	// Loop through each pixel in the image and set its value in the grayscale image
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			// Convert the color of the current pixel to grayscale and set it in the grayscale image
			grayImg.Set(x, y, color.GrayModel.Convert(img.At(x, y)))
		}
	}

	// Return the grayscale image
	return grayImg
}

// MonoChrome generates a monochrome (binary) version of a grayscale image.
// The threshold parameter is used to determine which pixels are converted to black and which are converted to white.
func MonoChrome(img *image.Gray, threshold uint8) *image.Gray {

	// Create a new grayscale image with the same bounds as the input image
	bounds := img.Bounds()
	grayImg := image.NewGray(bounds)

	// Loop through each pixel in the image and set its value in the monochrome image
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			// Get the grayscale value of the current pixel
			grayValue := img.GrayAt(x, y).Y

			// If the grayscale value is below the threshold, set the pixel to black (0)
			// Otherwise, set the pixel to white (255)
			if grayValue <= threshold {
				grayImg.SetGray(x, y, color.Gray{Y: 0})
			} else {
				grayImg.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}

	// Return the monochrome image
	return grayImg
}

// CutTheWhite removes the white border from a grayscale image by cropping it.
func CutTheWhite(img *image.Gray) *image.Gray {
	// Get the bounds of the input image
	rect := img.Bounds()

	// Initialize variables to keep track of the minimum and maximum x and y values
	minX, minY, maxX, maxY := rect.Max.X, rect.Max.Y, 0, 0

	// Loop through each pixel in the image and update the minimum and maximum x and y values as needed
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			// Check if the current pixel is black (0)
			grayColor := img.GrayAt(x, y).Y
			if grayColor == 0 {
				// Update the minimum and maximum x and y values accordingly
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	// Calculate the width and height of the new image
	width := maxX - minX + 1
	height := maxY - minY + 1

	// Create a new grayscale image and copy the pixels into it
	newImg := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			newImg.SetGray(x, y, img.GrayAt(x+minX, y+minY))
		}
	}

	return newImg
}

// MergeHorizontally merges two grayscale images horizontally.
// It returns the merged image and an error if the input images are not compatible.
func MergeHorizontally(img1, img2 *image.Gray) (*image.Gray, error) {

	// Check if input images are nil
	if img1 == nil || img2 == nil {
		return nil, errors.New("input images cannot be nil")
	}

	// Check if input images have equal heights
	if img1.Bounds().Dy() != img2.Bounds().Dy() {
		return nil, errors.New("input images must have equal heights")
	}

	// Get the width and height of the input images
	width1 := img1.Bounds().Dx()
	width2 := img2.Bounds().Dx()
	height := img1.Bounds().Dy()

	// Create a new grayscale image with the combined width and shared height,
	// and set the initial pixel values to white
	merged := image.NewGray(image.Rect(0, 0, width1+width2, height))

	// Iterate through the pixels of the input images
	for y := 0; y < height; y++ {
		// Copy pixels from the first image to the merged image
		for x := 0; x < width1; x++ {
			merged.Set(x, y, img1.GrayAt(x, y))
		}
		// Copy pixels from the second image to the merged image
		for x := 0; x < width2; x++ {
			merged.Set(x+width1, y, img2.GrayAt(x, y))
		}
	}

	// Return the merged image and no error
	return merged, nil
}

// FindLetterBoxes finds and segments characters in a captcha image.
// The maxLength parameter specifies the maximum allowed width of a single character.
func FindLetterBoxes(img *image.Gray, maxLength int) []image.Rectangle {

	// Get the dimensions of the input image
	width, height := img.Bounds().Dx(), img.Bounds().Dy()

	// Create a boolean array to keep track of which columns have black pixels
	colHasBlack := make([]bool, width)

	// Loop through each pixel in the image and update the colHasBlack array as needed
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			if img.GrayAt(x, y).Y == 0 {
				colHasBlack[x] = true
				break
			}
		}
	}

	// Initialize variables to keep track of letter boxes and the starting column of a potential letter
	letterBoxes := make([]image.Rectangle, 0)
	start := -1

	// Loop through each column of the image and create letter boxes as needed
	for x := 0; x < width; x++ {
		if colHasBlack[x] {
			// If this is the start of a potential letter, record its starting column
			if start == -1 {
				start = x
			}
		} else {
			// If this is the end of a potential letter, create a letter box and add it to the list of letter boxes
			if start != -1 {
				end := x - 1
				if end-start+1 <= maxLength {
					letterBoxes = append(letterBoxes, image.Rect(start, 0, end+1, height))
				} else {
					mid := (start + end) / 2
					letterBoxes = append(letterBoxes, image.Rect(start, 0, mid+1, height))
					letterBoxes = append(letterBoxes, image.Rect(mid+1, 0, end+1, height))
				}
				start = -1
			}
		}
	}

	// If a potential letter extends to the edge of the image, create a letter box and add it to the list of letter boxes
	if start != -1 {
		end := width - 1
		if end-start+1 <= maxLength {
			letterBoxes = append(letterBoxes, image.Rect(start, 0, end+1, height))
		} else {
			mid := (start + end) / 2
			letterBoxes = append(letterBoxes, image.Rect(start, 0, mid+1, height))
			letterBoxes = append(letterBoxes, image.Rect(mid+1, 0, end+1, height))
		}
	}

	// Return the list of letter boxes
	return letterBoxes
}

// ExtractFeatures extracts image features and returns a binary string.
func ExtractFeatures(img *image.Gray) (string, error) {
	// Get the dimensions of the input image
	bounds := img.Bounds()

	// Pre-allocate a byte slice with enough capacity for the binary string
	binaryStr := make([]byte, 0, bounds.Dx()*bounds.Dy())

	// Loop over each pixel in the image and append its binary value to the byte slice
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get the gray value of the current pixel
			pixel := img.GrayAt(x, y).Y

			// Append the binary value of the pixel to the byte slice
			if pixel == 0 {
				binaryStr = append(binaryStr, '1')
			} else {
				binaryStr = append(binaryStr, '0')
			}
		}
	}

	// Compress the binary string using zlib compression
	compressedData := new(bytes.Buffer)
	compressor, err := zlib.NewWriterLevel(compressedData, zlib.BestCompression)
	if err != nil {
		return "", err
	}
	_, err = compressor.Write(binaryStr)
	if err != nil {
		return "", err
	}
	err = compressor.Close()
	if err != nil {
		return "", err
	}

	// Return the hexadecimal string representation of the compressed binary data
	return hex.EncodeToString(compressedData.Bytes()), nil
}

// SaveGrayToPNG saves a grayscale image to a PNG file.
func SaveGrayToPNG(fileName string, img *image.Gray) error {
	// Create the output file
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// Encode the image as a PNG and write it to the output file
	err = png.Encode(file, img)
	if err != nil {
		return err
	}

	// Return nil to indicate success
	return nil
}
