from compiler.Compiler import compile_program
import argparse

parser = argparse.ArgumentParser()
parser.add_argument("input_file", type=str)
parser.add_argument("-o")
args = parser.parse_args()

if not args.o:
    print("No output file specified. Outputting to output.txt")
    args.o = "output.txt"

input_file = open(args.input_file, 'r')
prog = input_file.read()
input_file.close()

instructions = compile_program(prog)
output_file = open(args.o, 'w')
for insn in instructions:
    output_file.write(insn)
    output_file.write('\n')
output_file.close()





