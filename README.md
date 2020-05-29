# unconditional-mpc
unconditional-mpc is an MPC framework.

It contains a compiler, written in Python, for translating programs in a simple imperative language into a list of instructions.

It contains a runtime, written in Golang, which allows several parties to run the protocol specified by such a list of instructions. 

# Installation
First install Python and Go, and clone repository.

To run the compiler, the package 'ply' must be installed (first install pip, if not done already):
```bash
python -m pip install ply
```

# Running compiler
To compile a program, the general command is:
```bash
python compile_script.py input_path -o output_path
```
To test that this works, try running the following:
```bash
python compile_script.py example_programs/input.txt -o instructions.txt
```
This should result in the following appearing in instructions.txt:

```txt
INPUT 1 a
INPUT 2 b
PLUS a b sum
OUTPUT sum output
```

# Language guide
For a specification of the source language used for writing programs to the compiler, please see the project report pdf.
