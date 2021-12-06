package main

import (
	"encoding/json"
	"flag"
	"fmt"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/wbrown/novelai-research-tool/gpt-bpe"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

var encoder gpt_bpe.GPTEncoder

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

const (
	BiasString    BiasType = 0
	BiasTokens             = 1
	BiasLitString          = 2
)

type BiasSequences struct {
	Sequences []gpt_bpe.Tokens `json:"sequences"`
	Type      BiasType         `json:"type"`
}

type LoreBiasGroup struct {
	YamlPhrases          *[]string        `json:"-" yaml:"phrases"`
	Phrases              *[]BiasSequences `json:"phrases,omitempty" yaml:"-"`
	Bias                 *float64         `json:"bias,omitempty" yaml:"bias"`
	EnsureSequenceFinish *bool            `json:"ensure_sequence_finish,omitempty" yaml:"ensureSequenceFinish"`
	GenerateOnce         *bool            `json:"generate_once,omitempty" yaml:"generateOnce"`
	Enabled              *bool            `json:"enabled,omitempty" yaml:"enabled"`
	WhenInactive         *bool            `json:"whenInactive,omitempty" yaml:"whenInactive"`
}

type LoreBiasGroups []LoreBiasGroup

type LorebookEntry struct {
	Text                *string         `json:"text,omitempty" yaml:"text"`
	ContextCfg          *ContextConfig  `json:"contextConfig,omitempty" yaml:"contextConfig"`
	LastUpdatedAt       *int            `json:"lastUpdatedAt,omitempty" yaml:"lastUpdatedAt"`
	DisplayName         *string         `json:"displayName,omitempty" yaml:"displayName"`
	Keys                *[]string       `json:"keys,omitempty" yaml:"keys"`
	SearchRange         *int            `json:"searchRange,omitempty" yaml:"searchRange"`
	Enabled             *bool           `json:"enabled,omitempty" yaml:"enabled"`
	ForceActivation     *bool           `json:"forceActivation,omitempty" yaml:"forceActivation"`
	KeyRelative         *bool           `json:"keyRelative,omitempty" yaml:"keyRelative"`
	NonStoryActivatable *bool           `json:"nonStoryActivatable,omitempty" yaml:"nonStoryActivatable"`
	CategoryId          *string         `json:"category,omitempty" yaml:"categoryId"`
	LoreBiasGroups      *LoreBiasGroups `json:"loreBiasGroups,omitempty" yaml:"loreBiasGroups"`
}

type BiasType uint

type Category struct {
	Name                *string         `json:"name,omitempty" yaml:"name"`
	Id                  *string         `json:"id,omitempty" yaml:"id"`
	Enabled             *bool           `json:"enabled,omitempty" yaml:"enabled"`
	CreateSubcontext    *bool           `json:"createSubcontext,omitempty" yaml:"createSubcontext"`
	SubcontextSettings  *LorebookEntry  `json:"subcontextSettings,omitempty" yaml:"subcontextSettings"`
	UseCategoryDefaults *bool           `json:"useCategoryDefaults,omitempty" yaml:"useCategoryDefaults"`
	CategoryDefaults    *LorebookEntry  `json:"categoryDefaults,omitempty" yaml:"categoryDefaults"`
	LoreBiasGroups      *LoreBiasGroups `json:"loreBiasGroups,omitempty" yaml:"loreBiasGroups"`
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

func (biasGroups *LoreBiasGroups) RealizeBiases() {
	for biasIdx := range *biasGroups {
		biasGroup := (*biasGroups)[biasIdx]
		if biasGroup.YamlPhrases != nil {
			if (*biasGroups)[biasIdx].Phrases == nil {
				biasSequences := make([]BiasSequences, 0)
				(*biasGroups)[biasIdx].Phrases = &biasSequences
			}
			for phraseIdx := range *biasGroup.YamlPhrases {
				jsonifiedPhrase := BiasSequences{
					Sequences: make([]gpt_bpe.Tokens, 0),
					Type:      BiasLitString,
				}
				phraseString := (*biasGroup.YamlPhrases)[phraseIdx]
				tokens := encoder.Encode(&phraseString)
				jsonifiedPhrase.Sequences = append(jsonifiedPhrase.Sequences,
					*tokens)
				*(*biasGroups)[biasIdx].Phrases = append(
					*(*biasGroups)[biasIdx].Phrases, jsonifiedPhrase)
			}
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
	if category.LoreBiasGroups != nil {
		category.LoreBiasGroups.RealizeBiases()
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
			if entry.LoreBiasGroups != nil {
				entry.LoreBiasGroups.RealizeBiases()
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

func (lorebook *Lorebook) ToPlaintext() string {
	entryStrings := make([]string, 0)
	for entryIdx := range lorebook.Entries {
		entry := lorebook.Entries[entryIdx]
		if entry.Enabled == nil || *entry.Enabled {
			normalizedDisplayName := strings.Replace(*entry.DisplayName, ":", " -", -1)
			entryStrings = append(entryStrings,
				strings.Join([]string{normalizedDisplayName, *entry.Text}, ":\n"))
		}
	}
	return strings.Join(entryStrings, "\n***\n")
}

func init() {
	encoder = gpt_bpe.NewEncoder()
}

func main() {
	var outputFile string
	var plaintext bool
	flag.StringVar(&outputFile, "o", "output.lorebook", "output lorebook filename")
	flag.BoolVar(&plaintext, "p", false, "plaintext output")
	flag.Parse()
	inputFileArgs := flag.Args()
	if len(inputFileArgs) == 0 {
		fmt.Println("Usage: nsb [-o output-file] [-p] definition.yaml ...")
		flag.PrintDefaults()
		os.Exit(1)
	}
	// Windows requires us to do our own glob expansion.
	inputFiles := make([]string, 0)
	for inputArgIdx := range inputFileArgs {
		candidate := inputFileArgs[inputArgIdx]
		filePaths, err := filepath.Glob(candidate)
		if err != nil {
			log.Fatal(err)
		}
		inputFiles = append(inputFiles, filePaths...)
	}
	categories := make(CategoriesMap, 0)
	lorebook := Lorebook{
		Version: 3,
	}
	for fileIdx := range inputFiles {
		fileName := inputFiles[fileIdx]
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Fatal(fmt.Sprintf("Error reading %s:\n%v", fileName, err))
		}
		defs := Definition{}
		if err := yaml.Unmarshal(data, &defs); err != nil {
			log.Fatal(fmt.Sprintf("Error processing %s:\n%v", fileName, err))
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
	var outputBytes []byte
	if plaintext {
		text := lorebook.ToPlaintext()
		outputBytes = []byte(text)
	} else {
		var err error
		outputBytes, err = json.MarshalIndent(lorebook, "", " ")
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := ioutil.WriteFile(outputFile, outputBytes, 0755); err != nil {
		log.Fatal(err)
	}
}
