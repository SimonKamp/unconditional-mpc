import ply.lex as lex
import ply.yacc as yacc
from compiler import Lexer, Parser
from compiler.ASTmanipulations import ASTworker


def compile_program(prog):
    """Compilation process:
    1. Lex/Parse, and get the AST

    2. Confirm that the AST is a valid program:
    2-1. Check that there is a function named main with no parameters
    2-2. Check no input/output in other functions than main
    2-3. Check no recursion (+ no calls to main)
    2-4. Check function calls have correct number of arguments, and call existing function. Also distinct names.
    2-5. Check no use of undeclared variables
    2-6. Check output vars have distinct names
    2-7. Type check bool/num

    3. Perform AST-transformations to allow translation to intermediate representation
    3-1. Rewrite uminus
    3-2. Rewrite == to !=
    3-3. Change reused variable names
    3-4. Insert tmps
    3-5. Inline program, rewrite ifs to jumps
    3-6. Constant propagation

    4. Emit instructions in IR-language"""

    # Step 1
    lexer = lex.lex(module=Lexer)
    parser = yacc.yacc(module=Parser)
    lexer.error = False

    ast = parser.parse(prog)
    if lexer.error:
        return None
   # Step 2
    ast_worker = ASTworker(ast)

    # Step 2.1
    has_main = ast_worker.check_main()
    if not has_main:
        return None

    # Step 2.2
    found_io_error = ast_worker.check_illegal_input_output()
    if found_io_error:
        return None

    # Step 2.3
    try:
        has_recursion = ast_worker.check_recursion()
    except Exception as inst:
        for arg in inst.args:
            print(arg)
        return None
    if has_recursion:
        return None

    # Step 2.4
    has_bad_func_call = ast_worker.check_bad_func_calls()
    if has_bad_func_call:
        return None

    # Step 2.5
    uses_undeclared_var = ast_worker.check_undeclared_var()
    if uses_undeclared_var:
        return None

    # Step 2.6
    has_distinct_output_names = ast_worker.check_io_names()
    if not has_distinct_output_names:
        return None

    # Step 2.7
    is_type_sound = ast_worker.type_check_bool_num()
    if not is_type_sound:
        return None
    # Step 3
    ast_worker.rewrite_uminus()
    ast_worker.rewrite_eq()
    ast_worker.change_reused_var_names()
    ast_worker.smart_inline_program()
    ast_worker.insert_tmps()
    ast_worker.constant_propagation()
    ast_worker.type_check_bool_num(annotating_for_xor=True)
    ast_worker.introduce_xor()

    instructions = ast_worker.emit_instructions()
    return instructions


def compile_program_with_prints(prog):
    """Compilation process:
    1. Lex/Parse, and get the AST

    2. Confirm that the AST is a valid program:
    2-1. Check that there is a function named main with no parameters
    2-2. Check no input/output in other functions than main
    2-3. Check no recursion (+ no calls to main)
    2-4. Check function calls have correct number of arguments, and call existing function. Also distinct names.
    2-5. Check no use of undeclared variables
    2-6. Check output vars have distinct names
    2-7. Type check bool/num

    3. Perform AST-transformations to allow translation to intermediate representation
    3-1. Rewrite uminus
    3-2. Rewrite == to !=
    3-3. Change reused variable names
    3-4. Insert tmps
    3-5. Inline program, rewrite ifs to jumps
    3-6. Constant propagation

    4. Emit instructions in IR-language"""

    # Step 1
    lexer = lex.lex(module=Lexer)
    parser = yacc.yacc(module=Parser)
    lexer.error = False

    ast = parser.parse(prog)
    if lexer.error:
        return None

    print('Initial program:')
    print(ast.readable_str())

    # Step 2
    ast_worker = ASTworker(ast)


    # Step 2.1
    print("Checking for main function")
    has_main = ast_worker.check_main()
    if not has_main:
        return None

    # Step 2.2
    print("Checking for illegal I/O....")
    found_io_error = ast_worker.check_illegal_input_output()
    if found_io_error:
        return None

    # Step 2.3
    print("Checking for recursion....")
    try:
        has_recursion = ast_worker.check_recursion()
    except Exception as inst:
        for arg in inst.args:
            print(arg)
        return None
    if has_recursion:
        return None

    # Step 2.4
    print("Checking for bad func calls....")
    has_bad_func_call = ast_worker.check_bad_func_calls()
    if has_bad_func_call:
        return None

    # Step 2.5
    print("Checking for undeclared vars....")
    uses_undeclared_var = ast_worker.check_undeclared_var()
    if uses_undeclared_var:
        return None

    # Step 2.6
    print("Checking distinct I/O names....")
    has_distinct_output_names = ast_worker.check_io_names()
    if not has_distinct_output_names:
        return None

    # Step 2.7
    print("Type checking nums and booleans....")
    is_type_sound = ast_worker.type_check_bool_num()
    if not is_type_sound:
        return None
    # TODO: Type checking could have better error messages (include path)

    print("Checks done")

    # Step 3
    ast_worker.rewrite_uminus()
    print('-----------------------------------------')
    print('Removed Uminus:')
    print(ast_worker.prog.readable_str())

    ast_worker.rewrite_eq()
    print('-----------------------------------------')
    print('Removed EQUALS:')
    print(ast_worker.prog.readable_str())

    ast_worker.change_reused_var_names()
    print('-----------------------------------------')
    print('Changed reused variable names:')
    print(ast_worker.prog.readable_str())

    ast_worker.smart_inline_program()
    print('-----------------------------------------')
    print('Smart inline:')
    print(ast_worker.prog.readable_str())

    ast_worker.insert_tmps()
    print('-----------------------------------------')
    print('Inserted tmps:')
    print(ast_worker.prog.readable_str())

    ast_worker.constant_propagation()
    print('-----------------------------------------')
    print('Constant propagation:')
    print(ast_worker.prog.readable_str())

    ast_worker.type_check_bool_num(annotating_for_xor=True)
    ast_worker.introduce_xor()
    print('-----------------------------------------')
    print('Introduced XOR:')
    print(ast_worker.prog.readable_str())

    instructions = ast_worker.emit_instructions()
    print('-----------------------------------------')
    print('Instructions:')
    for insn in instructions:
        print(insn)



