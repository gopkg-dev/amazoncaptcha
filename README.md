# Amazon Captcha Solver

AmazonCaptcha is a Go library that provides a simple and easy-to-use API for solving text captchas used by Amazon. 
This library simplifies the process of captcha resolution and enables fast integration into your applications, making it a powerful tool for your development needs.

## Installation

To use this library, you need to have Go installed on your system. You can then install it using the following command:

```
go get github.com/gopkg-dev/amazoncaptcha
```

## Usage

Using this library is simple:

```go
package main

import (
	"fmt"
	"os"
	"github.com/gopkg-dev/amazoncaptcha"
)

func main() {
	file, err := os.Open("captcha.jpg")
	if err != nil {
		fmt.Printf("Error opening captcha file: %v", err)
		return
	}
	defer file.Close()

	result, err := amazoncaptcha.Solve(file)
	if err != nil {
		fmt.Printf("Error solving captcha: %v", err)
		return
	}

	fmt.Printf("Captcha solution: %s\n", result)
}

```

In this example, we load a captcha image from a file (`"captcha.jpg"`) and solve it using the default solver provided by this library. The result is printed to the console.

## Training

![Training](/doc/training.gif)

This library also includes a custom training tool for solving simple captchas. Our tool is designed to simplify the process of training machine learning models for captcha solving, by automating the collection and extraction of relevant features from large datasets.

With our training tool, you can:

- Automatically collect large volumes of captcha images from multiple sources
- Extract and preprocess relevant features from these images
- Train and optimize your own machine learning models

By using this tool, you can quickly create custom captcha solvers optimized for your specific use case.

Note: The use of our tool to exploit or misuse captchas in any way may be against the terms of service of websites that use them, and is not endorsed by this library or its developers.

# Testing

This project provides a set of tests to solve Amazon CAPTCHA problems. It includes the following tests:

1. `TestDownloadCaptchaImages`: Downloads CAPTCHAs from the Amazon CAPTCHA service and saves them to disk if the letters haven't appeared in the dataset before.
2. `TestSplitAndSaveCaptchaByLetter`: Splits multiple Amazon CAPTCHAs stored in a directory into individual letters and saves them to separate directories for subsequent machine learning modeling.
3. `TestExtractFeatures`: Calculates image features for each letter extracted from split images, then saves those features to JSON files for further use.
4. `TestSolveBatch`: Performs batch testing on multiple Amazon CAPTCHAs stored in a directory. Each CAPTCHA is solved using the `Solve` function and compared against the expected answer to determine its accuracy.

## Notes

- This project aims to provide a set of testing tools to solve Amazon CAPTCHA problems but does not guarantee the accuracy of the test results.
- When using this project, please be respectful of Amazon's terms of service and privacy policy.

## Acknowledgments

We would like to thank the open-source community for providing inspiration and resources for this project, including [a-maliarov/amazoncaptcha](https://github.com/a-maliarov/amazoncaptcha) which served as a reference for our development. We are grateful for their contributions to the field of captcha solving, and we hope that our library can help further advance this technology.
## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
