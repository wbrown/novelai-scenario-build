cmake_minimum_required(VERSION 3.14)
project(novelai-scenario-build)

## Golang target support
execute_Process(COMMAND go version
        RESULT_VARIABLE EXIT_CODE)
if (NOT ${EXIT_CODE} EQUAL 0)
    message(FATAL_ERROR
            "You need to have a `golang` environment installed with an appropriately set GOROOT.")
endif()

if(WIN32)
    set(NSB_BINARY "${PROJECT_SOURCE_DIR}/nsb.exe")
else()
    set(NSB_BINARY "${PROJECT_SOURCE_DIR}/nsb")
endif()

add_custom_command(
        OUTPUT "${NSB_BINARY}"
        WORKING_DIRECTORY "${PROJECT_SOURCE_DIR}"
        DEPENDS nsb.go
        COMMAND go build nsb.go)

macro(add_scenario_sources TARGET)
    foreach(_yaml_source ${ARGN})
        message("Will process ${_yaml_source} for ${TARGET}")
        list(APPEND ${TARGET}_SOURCES ${_yaml_source})
    endforeach()
endmacro()

macro(generate_artifacts TARGET OUTPUT)
    add_custom_command(
            OUTPUT "${PROJECT_SOURCE_DIR}/${OUTPUT}.scenario"
            OUTPUT "${PROJECT_SOURCE_DIR}/${OUTPUT}.lorebook"
            OUTPUT "${PROJECT_SOURCE_DIR}/${OUTPUT}.txt"
            WORKING_DIRECTORY "${PROJECT_SOURCE_DIR}"
            COMMAND ${NSB_BINARY} -p -o ${PROJECT_SOURCE_DIR}/${OUTPUT}
                ${${TARGET}_SOURCES}
            DEPENDS ${NSB_BINARY}
                    ${${TARGET}_SOURCES})
    list(APPEND GEN_TARGETS ${PROJECT_SOURCE_DIR}/${OUTPUT}.scenario
                            ${PROJECT_SOURCE_DIR}/${OUTPUT}.lorebook
                            ${PROJECT_SOURCE_DIR}/${OUTPUT}.txt)
endmacro()

macro(build_artifacts)
    add_custom_target( generated ALL DEPENDS ${GEN_TARGETS})
endmacro()

## User editable parts below.

add_scenario_sources(darkest-dungeon
        examples/darkest-dungeon/scenario.yaml
        examples/darkest-dungeon/meta.yaml
        examples/darkest-dungeon/preamble.yaml
        examples/darkest-dungeon/classes-add.yaml
        examples/darkest-dungeon/classes_D.yaml
        examples/darkest-dungeon/classes_F.yaml
        examples/darkest-dungeon/classes_M.yaml
        examples/darkest-dungeon/classes_Y.yaml
        examples/darkest-dungeon/factions-add.yaml
        examples/darkest-dungeon/factions.yaml
        examples/darkest-dungeon/locations-add.yaml
        examples/darkest-dungeon/locations.yaml
        examples/darkest-dungeon/monsters.yaml)

add_scenario_sources(white-samurai-and-ronin
        examples/feudal-edo/kyoto-lorebook.yaml
        examples/feudal-edo/white-samurai-and-ronin.yaml)

add_scenario_sources(white-samurai-and-courtesan
        examples/feudal-edo/kyoto-lorebook.yaml
        examples/feudal-edo/white-samurai-and-the-courtesan.yaml)

add_scenario_sources(courtesan-in-feudal-kyoto
        examples/feudal-edo/kyoto-lorebook.yaml
        examples/feudal-edo/courtesan-in-feudal-kyoto.yaml)

generate_artifacts(darkest-dungeon
    examples/darkest-dungeon/darkest-dungeon)
generate_artifacts(white-samurai-and-ronin
    "examples/feudal-edo/The White Samurai and the Ronin")
generate_artifacts(courtesan-in-feudal-kyoto
    "examples/feudal-edo/The Courtesan in Feudal Kyoto")
generate_artifacts(white-samurai-and-courtesan
    "examples/feudal-edo/The White Samurai and the Courtesan")

build_artifacts()


