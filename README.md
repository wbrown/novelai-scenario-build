NovelAI Scenario Builder
========================
This is a tool designed to make your large scenario or lorebook project easier to work with and collaborate on with other people.

It consumes one or more YAML format files that hold definitions and produces NovelAI files that can be imported directly into NovelAI.

The reference example is `examples/darkest-dungeon` and contains examples of:
* Phrase Biasing
* Lorebook Keys (with and without regex)
* Category settings
* Advanced features hidden from the NovelAI UI
* Scenario settings.

Partially working:
* A/N, and Memory configuration -- you need to make sure your A/N and memory are set *after* the context configuration for these.

The current unimplemented gaps pending addressing are:
* Ephemeral context (who really uses these?)
* Modules (you can just select on import)
* Presets (select on import)

Below is an example entry that shows the most essential fields.
```yaml
categories:
  Monsters:
    createSubcontext: true
    subcontextSettings:
      contextConfig:
        suffix: "\n\n"
        tokenBudget: 2048
        reservedTokens: 200
        budgetPriority: -200
        insertionPosition: 0
lorebook:
  - category: Monsters
    config:
      contextConfig:
        suffix: "\n"
        tokenBudget: 2048
        budgetPriority: -200
        TrimDirection: doNotTrim
    entries:
      "Undead: Bone Rabble":
        keys:
          - Bone Rabble
          - Bone Rabbles
        text: >
          Bone Rabbles are the weakest kind of skeletal enemies you may
          encounter. They are barely held together by the Necromancer's magic.

          Bone Rabbles' main purpose in combat is to use their bodies to
          protect their allies. Bump in the Night is a weak attack, where
          the Bone Rabble clobbers its target with a makeshift club.
          Tic-Toc closes the gap between the Bone Rabble and its target.

          [ Bone Rabble Abilities: Bump in the Night, Tic-Toc; Bone Rabble
          Equipment: Club, Light Armor ]
        loreBiasGroups:
          - phrases:
              - " Rabble"
            bias: -0.1
          - phrases:
              - " Rabble"
            bias: 0
            whenInactive: true
          - phrases:
              - " Bump in the Night"
              - " Tic-Toc"
            bias: 0.1
```

A Windows executable is provided, `nsb.exe`, in the repository if you do not want to build it yourself on Windows.

Invocation is simple:
* `nsb.exe -o darkest-dungeon.lorebook examples/darkest-dungeon/*.yaml`

This will create a `darkest-dungeon.lorebook` file out of all the YAML definitions in the `examples/darkest-dungeon` directory.

It also accepts a `-p` option to generate text files suitable for module training:
* `nsb.exe -p -o darkest-dungeon.txt examples/darkest-dungeon/*.yaml`

Binary Build Instructions
-------------------------
* Ensure that the following are installed on your machine:
   - `golang`
   - `cmake`
* Run `cmake .`
* Run `make`
