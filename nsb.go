package main

import (
	"encoding/json"
	"flag"
	"fmt"
	uuid "github.com/nu7hatch/gouuid"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"
)

type Lorebook struct {
	Version    int             `json:"lorebookVersion"`
	Entries    []LorebookEntry `json:"entries,omitempty"`
	Categories []Category      `json:"categories,omitempty"`
	Settings   *struct {
		orderByKeyLocations bool
	} `json:"settings,omitempty"`
}

type ContextConfig struct {
	Prefix            *string `json:"prefix,omitempty" yaml:"prefix"`
	Suffix            *string `json:"suffix,omitempty" yaml:"suffix"`
	TokenBudget       *int    `json:"tokenBudget,omitempty" yaml:"tokenBudget"`
	ReservedTokens    *int    `json:"reservedTokens,omitempty" yaml:"reservedTokens"`
	BudgetPriority    *int    `json:"budgetPriority,omitempty" yaml:"budgetPriority"`
	TrimDirection     *string `json:"trimDirection,omitempty" yaml:"trimDirection"`
	InsertionType     *string `json:"insertionType,omitempty" yaml:"insertionType"`
	MaximumTrimType   *string `json:"maximumTrimType,omitempty" yaml:"maximumTrimType"`
	InsertionPosition *int    `json:"insertionPosition,omitempty" yaml:"insertionPosition"`
	Force             *bool   `json:"forced,omitempty" yaml:"forced"`
}

type LorebookEntry struct {
	Text                *string        `json:"text,omitempty" yaml:"text"`
	ContextCfg          *ContextConfig `json:"contextConfig,omitempty" yaml:"contextConfig"`
	LastUpdatedAt       *int           `json:"lastUpdatedAt,omitempty" yaml:"lastUpdatedAt""`
	DisplayName         *string        `json:"displayName,omitempty" yaml:"displayName"`
	Keys                *[]string      `json:"keys,omitempty" yaml:"keys"`
	SearchRange         *int           `json:"searchRange,omitempty" yaml:"searchRange"`
	Enabled             *bool          `json:"enabled,omitempty" yaml:"enabled"`
	ForceActivation     *bool          `json:"forceActivation,omitempty" yaml:"forceActivation"`
	KeyRelative         *bool          `json:"keyRelative,omitempty" yaml:"keyRelative"`
	NonStoryActivatable *bool          `json:"nonStoryActivatable,omitempty" yaml:"nonStoryActivatable"`
	CategoryId          *string        `json:"category,omitempty" yaml:"categoryId"`
}

type Category struct {
	Name                *string        `json:"name,omitempty" yaml:"name"`
	Id                  *string        `json:"id,omitempty" yaml:"id"`
	Enabled             *bool          `json:"enabled,omitempty" yaml:"enabled"`
	CreateSubcontext    *bool          `json:"createSubcontext,omitempty" yaml:"createSubcontext"`
	SubcontextSettings  *LorebookEntry `json:"subcontextSettings,omitempty" yaml:"subcontextSettings"`
	UseCategoryDefaults *bool          `json:"useCategoryDefaults,omitempty" yaml:"useCategoryDefaults"`
	CategoryDefaults    *LorebookEntry `json:"categoryDefaults,omitempty" yaml:"categoryDefaults"`
}

type Definition struct {
	Categories map[string]*Category
	Lorebook   []struct {
		Category *string
		Config   *LorebookEntry
		Entries  map[string]LorebookEntry
	}
}

type CategoriesMap map[string]*Category

func (defaults *LorebookEntry) RealizeDefaults(entry *LorebookEntry) {
	fields := reflect.TypeOf(*defaults)
	for field := 0; field < fields.NumField(); field++ {
		fieldValues := reflect.ValueOf(defaults).Elem().Field(field)
		if fieldValues.IsNil() {
			continue
		}
		entryValue := reflect.ValueOf(entry).Elem().Field(field)
		if entryValue.IsNil() {
			entryValue.Set(fieldValues)
		}
	}
}

func (categories *CategoriesMap) RealizeCategory(name string, category *Category) *Category {
	if lookup, ok := (*categories)[name]; ok {
		return lookup
	}
	if category == nil {
		newCategory := Category{
			Name: &name,
		}
		category = &newCategory
	} else if category.Name == nil {
		category.Name = &name
	}
	if category.Id == nil {
		categoryUuid, _ := uuid.NewV4()
		uuidStr := categoryUuid.String()
		category.Id = &uuidStr
	}
	(*categories)[name] = category
	return category
}

func (def *Definition) RealizeDefinition(categories *CategoriesMap) {
	for categoryKey := range def.Categories {
		categories.RealizeCategory(categoryKey, def.Categories[categoryKey])
	}
	for lorebookGroupIdx := range def.Lorebook {
		lorebookGroup := def.Lorebook[lorebookGroupIdx]
		var category *Category
		if lorebookGroup.Category != nil {
			category = categories.RealizeCategory(*lorebookGroup.Category, nil)
		}
		for entryKey := range lorebookGroup.Entries {
			entry := lorebookGroup.Entries[entryKey]
			entryName := entryKey
			entry.DisplayName = &entryName
			*entry.Text = strings.TrimSuffix(*entry.Text, "\n")
			if category != nil && entry.CategoryId == nil {
				entry.CategoryId = category.Id
			}
			if lorebookGroup.Config != nil {
				lorebookGroup.Config.RealizeDefaults(&entry)
			}
			lorebookGroup.Entries[entryName] = entry
		}
	}
}

func (def *Definition) ToJson(categories *CategoriesMap) []byte {
	lorebook := Lorebook{
		Version:    3,
		Entries:    make([]LorebookEntry, 0),
		Categories: make([]Category, 0),
	}
	for lorebookGroupIdx := range def.Lorebook {
		lorebookGroup := def.Lorebook[lorebookGroupIdx]
		for entryName := range lorebookGroup.Entries {
			lorebook.Entries = append(lorebook.Entries, lorebookGroup.Entries[entryName])
		}
	}
	for categoryKey := range *categories {
		lorebook.Categories = append(lorebook.Categories, *(*categories)[categoryKey])
	}
	jsonBytes, err := json.MarshalIndent(lorebook, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	return jsonBytes
}

func main() {
	var outputFile string
	flag.StringVar(&outputFile, "o", "output.lorebook", "output lorebook filename")
	flag.Parse()
	inputFiles := flag.Args()
	if len(inputFiles) == 0 {
		fmt.Println("Usage: nonflags.go [-o lorebook-file] definition.yaml ...")
		flag.PrintDefaults()
		os.Exit(1)
	}

	categories := make(CategoriesMap, 0)
	lorebook := Lorebook{
		Version: 3,
	}
	for fileIdx := range inputFiles {
		fileName := inputFiles[fileIdx]
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Fatal(err)
		}
		defs := Definition{}
		if err := yaml.Unmarshal(data, &defs); err != nil {
			log.Fatal(err)
		}
		defs.RealizeDefinition(&categories)
		for lorebookGroupIdx := range defs.Lorebook {
			lorebookGroup := defs.Lorebook[lorebookGroupIdx]
			for entryName := range lorebookGroup.Entries {
				lorebook.Entries = append(lorebook.Entries, lorebookGroup.Entries[entryName])
			}
		}
	}
	for categoryKey := range categories {
		lorebook.Categories = append(lorebook.Categories, *(categories)[categoryKey])
	}
	jsonBytes, err := json.MarshalIndent(lorebook, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	ioutil.WriteFile(outputFile, jsonBytes, 0755)
}
