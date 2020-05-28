# unconditional-mpc
unconditional-mpc is an MPC framework.

It contains a compiler, written in Python, for translating programs in a simple imperative language into a list of instructions.

It contains a runtime, written in Golang, which allows several parties to run the protocol specified by such a list of instructions. 

# Installation
First install Python and Go, and clone repository.

To run the compiler, the package 'ply' must be installed by running one of the following commands:
```bash
pip install ply
python -m pip install ply
```
