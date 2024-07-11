/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	backupoperatoriov1 "backup-operator.io/api/v1"
)

func main() {
	// Read the README.tpl.md template file
	tplContent, err := os.ReadFile("README.tpl.md")
	if err != nil {
		fmt.Printf("Error reading template file: %v\n", err)
		return
	}

	// Parse the template content
	tpl, err := template.New("readme").Parse(string(tplContent))
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		return
	}

	// Get the annotations and generate the markdown table
	annotations := backupoperatoriov1.ListAnnotations()
	markdownTable := annotations.String()

	// Create a data structure to pass to the template
	data := struct {
		Annotations string
	}{
		Annotations: markdownTable,
	}

	// Execute the template with the data
	var output bytes.Buffer
	if err := tpl.Execute(&output, data); err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		return
	}

	// Write the output to README.md
	if err := os.WriteFile("README.md", output.Bytes(), 0644); err != nil {
		fmt.Printf("Error writing output file: %v\n", err)
		return
	}

	fmt.Println("README.md generated successfully.")
	return
}
