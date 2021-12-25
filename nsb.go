package main

import (
	"encoding/json"
	"flag"
	"fmt"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/wbrown/novelai-research-tool/gpt-bpe"
	"github.com/wbrown/novelai-research-tool/novelai-api"
	"github.com/wbrown/novelai-research-tool/scenario"
	"github.com/wbrown/novelai-research-tool/structs"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var encoder gpt_bpe.GPTEncoder

type CategoriesMap map[string]*scenario.Category

func (categories *CategoriesMap) RealizeCategory(name string,
	category *scenario.Category) *scenario.Category {
	if lookup, ok := (*categories)[name]; ok {
		return lookup
	}
	if category == nil {
		newCategory := scenario.Category{
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

type Definition struct {
	Title             *string                    `yaml:"title"`
	Description       *string                    `yaml:"description"`
	Tags              *[]string                  `yaml:"tags"`
	Prompt            *string                    `yaml:"prompt"`
	Memory            *string                    `yaml:"memory"`
	AuthorsNote       *string                    `yaml:"authorsNote"`
	ModuleFile        *string                    `yaml:"moduleFile"`
	MemoryConfig      *scenario.ContextConfig    `yaml:"memoryConfig"`
	AuthorsNoteConfig *scenario.ContextConfig    `yaml:"authorsNoteConfig"`
	StoryConfig       *scenario.ContextConfig    `yaml:"storyConfig"`
	LorebookSettings  *scenario.LorebookSettings `yaml:"lorebookSettings"`
	Placeholders      *scenario.Placeholders     `yaml:"placeholders"`
	Categories        *CategoriesMap             `yaml:"categories"`
	Biases            *structs.BiasGroups        `yaml:"biases"`
	Lorebook          []struct {
		Category *string
		Config   *scenario.LorebookEntry
		Entries  map[string]scenario.LorebookEntry
	}
}

func RealizeScenario(sc *scenario.Scenario, inputFiles []string) {
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
		if defs.Title != nil {
			if sc.Title != "" {
				log.Printf("WARNING: Scenario title already set! "+
					"Overwriting with '%s'.", *defs.Title)
			}
			sc.Title = *defs.Title
		}
		if defs.Description != nil {
			if sc.Title != "" {
				log.Printf("WARNING: Scenario description already"+
					"set! Overwriting with '%s'.", *defs.Description)
			}
			sc.Description = *defs.Description
		}
		if defs.Prompt != nil {
			if sc.Prompt != "" {
				log.Printf("WARNING: Prompt already set! Overwriting.")
			}
			sc.Prompt = *defs.Prompt
		}
		if defs.Memory != nil {
			if sc.Context[0].Text != nil {
				log.Printf("WARNING: Memory already set! Overwriting.")
			}
			sc.Context[0].Text = defs.Memory
		}
		if defs.AuthorsNote != nil {
			if sc.Context[1].Text != nil {
				log.Printf("WARNING: Author's Note already set! " +
					"Overwriting.")
			}
			sc.Context[1].Text = defs.AuthorsNote
		}
		if defs.Tags != nil {
			for tagIdx := range *defs.Tags {
				newTag := (*defs.Tags)[tagIdx]
				exists := false
				for oldTagsIdx := range sc.Tags {
					if sc.Tags[oldTagsIdx] == newTag {
						exists = true
						break
					}
				}
				if !exists {
					sc.Tags = append(sc.Tags, newTag)
				}
			}
		}
		if defs.MemoryConfig != nil {
			if len(sc.Context) < 1 {
				sc.Context = append(sc.Context,
					scenario.ContextEntry{})
			}
			if sc.Context[0].ContextCfg != nil {
				log.Printf("WARNING: memoryConfig already set! " +
					"Overwriting.")
			}
			sc.Context[0].ContextCfg = defs.MemoryConfig
		}
		if defs.AuthorsNoteConfig != nil {
			if len(sc.Context) < 1 {
				sc.Context = append(sc.Context,
					scenario.ContextEntry{})
			}
			if len(sc.Context) < 1 {
				sc.Context = append(sc.Context,
					scenario.ContextEntry{})
			}
			if sc.Context[1].ContextCfg != nil {
				log.Printf("WARNING: authorsNoteConfig already set! " +
					"Overwriting.")
			}
			sc.Context[1].ContextCfg = defs.AuthorsNoteConfig
		}
		if defs.StoryConfig != nil {
			if sc.StoryContextConfig != nil {
				log.Printf("WARNING: authorsNoteConfig already set! " +
					"Overwriting.")
			}
			sc.StoryContextConfig = defs.StoryConfig
		}
		if defs.Placeholders != nil {
			if sc.PlaceholderMap == nil {
				sc.PlaceholderMap = make(scenario.Placeholders, 0)
			}
			sc.PlaceholderMap.Add(*defs.Placeholders)
		}
		if defs.Biases != nil {
			defs.Biases.RealizeBiases()
			if sc.Settings.Parameters.LogitBiasGroups == nil {
				sc.Settings.Parameters.LogitBiasGroups = defs.Biases
			} else {
				*sc.Settings.Parameters.LogitBiasGroups =
					append(*sc.Settings.Parameters.LogitBiasGroups,
						*defs.Biases...)
			}
		}
	}
	if len(sc.PlaceholderMap) > 0 {
		sc.PlaceholderMap.Realize()
		sc.Placeholders = make([]scenario.Placeholder, 0)
		for key := range sc.PlaceholderMap {
			sc.Placeholders = append(sc.Placeholders,
				*sc.PlaceholderMap[key])
		}
	}
}

func (def *Definition) RealizeLorebookDefs(categories *CategoriesMap) {
	if def.Categories == nil {
		def.Categories = &CategoriesMap{}
	}
	for categoryKey := range *def.Categories {
		categories.RealizeCategory(categoryKey, (*def.Categories)[categoryKey])
	}
	for lorebookGroupIdx := range def.Lorebook {
		lorebookGroup := def.Lorebook[lorebookGroupIdx]
		var category *scenario.Category
		if lorebookGroup.Category != nil {
			category = categories.RealizeCategory(*lorebookGroup.Category,
				nil)
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
	lorebook := scenario.Lorebook{
		Version:    3,
		Entries:    make([]scenario.LorebookEntry, 0),
		Categories: make([]scenario.Category, 0),
	}
	for lorebookGroupIdx := range def.Lorebook {
		lorebookGroup := def.Lorebook[lorebookGroupIdx]
		for entryName := range lorebookGroup.Entries {
			lorebook.Entries = append(lorebook.Entries,
				lorebookGroup.Entries[entryName])
		}
	}
	for categoryKey := range *categories {
		lorebook.Categories = append(lorebook.Categories,
			*(*categories)[categoryKey])
	}
	jsonBytes, err := json.MarshalIndent(lorebook, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	return jsonBytes
}

func init() {
	encoder = gpt_bpe.NewEncoder()
}

func RealizeLorebook(lorebook *scenario.Lorebook, categories *CategoriesMap,
	inputFiles []string) {
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
		defs.RealizeLorebookDefs(categories)
		for lorebookGroupIdx := range defs.Lorebook {
			lorebookGroup := defs.Lorebook[lorebookGroupIdx]
			for entryName := range lorebookGroup.Entries {
				lorebook.Entries = append(lorebook.Entries,
					lorebookGroup.Entries[entryName])
			}
		}
		if defs.LorebookSettings != nil {
			lorebook.Settings = *defs.LorebookSettings
		}
	}
	for categoryKey := range *categories {
		category := (*categories)[categoryKey]
		lorebook.Categories = append(lorebook.Categories,
			*category)
	}
}

func main() {
	var outputFile string
	var plaintext bool
	flag.StringVar(&outputFile, "o", "output", "output base filename")
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
	lorebook := scenario.Lorebook{
		Version: 3,
	}

	RealizeLorebook(&lorebook, &categories, inputFiles)
	scenario := scenario.Scenario{
		ScenarioVersion: 1,
		Lorebook:        lorebook,
		Context: []scenario.ContextEntry{
			scenario.ContextEntry{},
			scenario.ContextEntry{}},
		Settings: scenario.ScenarioSettings{
			Parameters: &novelai_api.NaiGenerateParams{},
		},
	}
	RealizeScenario(&scenario, inputFiles)

	var outputBytes []byte
	if plaintext {
		lorebook.ToPlaintextFile(outputFile + ".txt")
	}
	lorebook.ToFile(outputFile + ".lorebook")
	var err error
	outputBytes, err = json.MarshalIndent(scenario, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile(outputFile+".scenario", outputBytes,
		0755); err != nil {
		log.Fatal(err)
	}
}
