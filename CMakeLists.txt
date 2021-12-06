cmake_minimum_required(VERSION 3.14)
project(novelai-scenario-build)

## Golang target support
execute_Process(COMMAND go version
        RESULT_VARIABLE EXIT_CODE)
if (NOT ${EXIT_CODE} EQUAL 0)
    message(FATAL_ERROR
            "You need to have a `golang` environment installed with an appropriately set GOROOT.")
endif()

set(NSB_BINARY "${PROJECT_SOURCE_DIR}/nsb")

add_custom_command(
        OUTPUT "${NSB_BINARY}"
        WORKING_DIRECTORY "${PROJECT_SOURCE_DIR}"
        COMMAND go build nsb.go)

macro(add_scenario_sources TARGET)
    foreach(_yaml_source ${ARGN})
        message("Will process ${_yaml_source} for ${TARGET}")
        list(APPEND ${TARGET}_SOURCES ${_yaml_source})
    endforeach()
endmacro()

macro(generate_lorebook TARGET OUTPUT)
    add_custom_command(
            OUTPUT "${PROJECT_SOURCE_DIR}/${OUTPUT}"
            WORKING_DIRECTORY "${PROJECT_SOURCE_DIR}"
            COMMAND ${NSB_BINARY} -o ${PROJECT_SOURCE_DIR}/${OUTPUT}
                ${${TARGET}_SOURCES})
    list(APPEND GEN_TARGETS ${PROJECT_SOURCE_DIR}/${OUTPUT})
endmacro()

macro(generate_plaintext TARGET OUTPUT)
    add_custom_command(
            OUTPUT "${PROJECT_SOURCE_DIR}/${OUTPUT}"
            WORKING_DIRECTORY "${PROJECT_SOURCE_DIR}"
            COMMAND ${NSB_BINARY} -p -o ${PROJECT_SOURCE_DIR}/${OUTPUT}
                ${${TARGET}_SOURCES})
    list(APPEND GEN_TARGETS ${PROJECT_SOURCE_DIR}/${OUTPUT})
endmacro()

macro(build_artifacts)
    add_custom_target( generated ALL DEPENDS ${GEN_TARGETS})
endmacro()

## User editable parts below.

add_scenario_sources(darkest-dungeon
        examples/darkest-dungeon/preamble.yaml
        examples/darkest-dungeon/bosses.yaml
        examples/darkest-dungeon/classes.yaml
        examples/darkest-dungeon/factions.yaml
        examples/darkest-dungeon/monsters.yaml)

generate_lorebook(darkest-dungeon darkest-dungeon.lorebook)
generate_plaintext(darkest-dungeon darkest-dungeon.txt)
build_artifacts()


