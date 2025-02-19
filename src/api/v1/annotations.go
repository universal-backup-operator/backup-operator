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

package v1

import (
	"fmt"
	"sort"
	"strings"
)

type Annotation struct {
	Description string
	Name        string
}

type (
	ClassAnnotations []Annotation
	Annotations      map[string]ClassAnnotations
)

func (a *Annotations) String() string {
	var sb strings.Builder

	// Получаем и сортируем ключи (имена классов)
	classNames := make([]string, 0, len(*a))
	for className := range *a {
		classNames = append(classNames, className)
	}
	sort.Strings(classNames)

	// Итерация по отсортированным классам
	for _, className := range classNames {
		annotations := (*a)[className]

		// Сортируем аннотации внутри класса по имени
		sort.SliceStable(annotations, func(i, j int) bool {
			return annotations[i].Name < annotations[j].Name
		})

		sb.WriteString(fmt.Sprintf("\n### %s\n\n", className))
		sb.WriteString("| Name | Description |\n")
		sb.WriteString("|-------------|------------|\n")

		for _, annotation := range annotations {
			sb.WriteString(fmt.Sprintf("| `%s` | %s |\n", annotation.Name, annotation.Description))
		}
	}

	return strings.TrimSpace(sb.String())
}

// ListAnnotations lists all annotations available across all CRDs
func ListAnnotations() Annotations {
	return Annotations{
		"BackupRun": ClassAnnotations{
			{
				Description: "Set to any value and BackupSchedule won't delete this run during the rotation",
				Name:        AnnotationKeepBackupRun,
			},
			{
				Description: "It is set by operator after the restoration is completed successfully",
				Name:        AnnotationRestoredAt,
			},
			{
				Description: "Set to any value in case if you want to restore the backup",
				Name:        AnnotationRestore,
			},
		},
		"BackupSchedule": ClassAnnotations{
			{
				Description: "Can be set to any value to trigger schedule manually",
				Name:        AnnotationTriggerSchedule,
			},
		},
		"BackupStorage": ClassAnnotations{
			{
				Description: "Is set automatically and prevents accidental storage deletion",
				Name:        AnnotationDeletionProtection,
			},
		},
	}
}
