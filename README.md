# tg [![builds.sr.ht status](https://builds.sr.ht/~delthas/tg.svg)](https://builds.sr.ht/~delthas/tg?)

Filter stdout and stderr of a process with simple regex rules.

## Usage

You want to run `$ bar arg1 arg2 arg3`, but it outputs a lot of unnecessary lines:
```shell
$ bar arg1 arg2 arg3
Barring the foo... (0)
Barring the foo... (1)
Barring the foo... (2)
Barring the foo... (3)
IMPORTANT MESSAGE!!11!
Barring the foo... (4)
Barring the foo... (5)
```

You wish to run exactly the program exactly the same, but simply filter out specific line patterns.

Run:
```shell
$ tg -o 'Barring the foo.*' bar arg1 arg2 arg3
IMPORTANT MESSAGE!!11!
```

Syntax: `tg [[rule] ...] program [args]`

Each rule is one of:
- `-o <pattern>`: filter out any stdout line matching `pattern`
- `-e <pattern>`: filter out any stderr line matching `pattern`
- `-a <pattern>`: filter out any stdout or stderr line matching `pattern`
- `-O <pattern>`: only output stdout lines matching `pattern`
- `-E <pattern>`: only output stderr lines matching `pattern`
- `-A <pattern>`: only output stdout or stderr lines matching `pattern`

## Builds

| OS | tg |
|---|---|
| Linux x64 | [link](https://delthas.fr/tg/linux/tg) |
| Mac OS X x64 | [link](https://delthas.fr/tg/mac/tg) |
| Windows x64 | [link](https://delthas.fr/tg/windows/tg.exe) |
