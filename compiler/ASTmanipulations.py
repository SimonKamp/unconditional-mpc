from compiler import ASTnodes
from copy import deepcopy
from compiler.CallGraph import CallGraph


class ASTworker:
    """
    This class contains methods for transforming the AST output of the parser
    into an AST that can be translated to a sequence of instructions.

    Methods should be called in the right order, since some methods assume
    that certain transformations have already been completed, meaning
    that it is not necessary to consider e.g. IF-statements after those
    have already been removed.

    The correct order of transformation would be:
    1. transform_ifs
    2. remove_uminus
    3. insert_tmps
    """

    def __init__(self, prog):
        self.tmp_counter = 1
        self.func_call_counter = 1
        self.program_point_counter = 1
        self.var_redef_counter = 1
        self.private_cond_depth = 0
        self.prog = prog
        self.func_dict = {}
        for func in prog.funcs:
            self.func_dict[func.id.name] = func

    def check_main(self):
        found_main = False
        for func in self.prog.funcs:
            if func.id.name == 'main':
                if len(func.args) > 0:  # Not really necessary to have this anymore after we changed the grammar
                    print("ERROR in line %d: Function 'main' should not take any arguments." % func.lineno)
                    return False
                else:
                    found_main = True
        if not found_main:
            print("ERROR: No function called 'main' in program.")
            return False
        return True

    def check_illegal_input_output(self):
        found_error = False
        for func in self.prog.funcs:
            for stm in func.body.stms:
                if func.id.name != 'main':
                    if isinstance(stm, ASTnodes.InputStm):
                        found_error = True
                        print(
                            "ERROR in line %d: Found illegal input statement '%s' in function '%s'. Input statements may only be used in function 'main'." % (
                            stm.lineno, stm.readable_str(), func.id.name))
                    elif isinstance(stm, ASTnodes.OutputStm):
                        found_error = True
                        print(
                            "ERROR in line %d: Found illegal output statement '%s' in function '%s'. Output statements may only be used in function 'main'." % (
                            stm.lineno, stm.readable_str(), func.id.name))

        return found_error

    def check_recursion(self):
        cg = CallGraph(self.prog)
        return cg.func_overload or cg.has_recursion()

    def check_bad_func_calls(self):
        found_bad_func_call = False
        for func in self.prog.funcs:
            for stm in func.body.stms:
                if not isinstance(stm, ASTnodes.AssignStm):
                    continue
                if self.expr_has_bad_func_call(stm.expr):
                    found_bad_func_call = True
            if func.id.name != 'main' and self.expr_has_bad_func_call(func.body.expr):
                found_bad_func_call = True

        return found_bad_func_call

    def expr_has_bad_func_call(self, expr):
        if isinstance(expr, ASTnodes.Number) or isinstance(expr, ASTnodes.Boolean) or isinstance(expr, ASTnodes.Identifier):
            return False
        if isinstance(expr, ASTnodes.IfExpr):
            return self.expr_has_bad_func_call(expr.cond) or \
                   self.expr_has_bad_func_call(expr.then_branch) or \
                   self.expr_has_bad_func_call(expr.then_branch)
        if isinstance(expr, ASTnodes.Uminus) or isinstance(expr, ASTnodes.Not) or isinstance(expr, ASTnodes.LeakExpr):
            return self.expr_has_bad_func_call(expr.sub_expr)
        if isinstance(expr, ASTnodes.Binop):
            return self.expr_has_bad_func_call(expr.left) or self.expr_has_bad_func_call(expr.right)
        if not isinstance(expr, ASTnodes.FuncCallExpr):
            print("Found unexpected expression during check for bad func calls")
            raise SyntaxError

        if expr.func_id.name == 'randomnum' or expr.func_id.name == 'randombit':
            return False
        bad_call = False
        arguments_supplied = len(expr.args)
        arguments_expected = len(self.func_dict[expr.func_id.name].args)
        if arguments_supplied != arguments_expected:
            print("ERROR in line %d: Function call to '%s' has wrong number of arguments (%d). Expected %d." % (
            expr.lineno, expr.func_id.name, arguments_supplied, arguments_expected))
            bad_call = True

        return bad_call

    def check_undeclared_var(self):
        found_use_of_undeclared_var = False
        for func in self.prog.funcs:
            declared = set()
            for arg in func.args:
                declared.add(arg.name)
            for stm in func.body.stms:
                if isinstance(stm, ASTnodes.InputStm):
                    declared.add(stm.var.name)
                elif isinstance(stm, ASTnodes.OutputStm):
                    if stm.output_var.name not in declared:
                        found_use_of_undeclared_var = True
                        print("ERROR in line %d: Use of undeclared variable '%s'." % (stm.lineno, stm.output_var.name))
                elif isinstance(stm, ASTnodes.AssignStm):
                    if self.expr_uses_undeclared_var(stm.expr, declared, func.id.name):
                        found_use_of_undeclared_var = True
                    declared.add(stm.var.name)
            if func.id.name != 'main' and self.expr_uses_undeclared_var(func.body.expr, declared, func.id.name):
                found_use_of_undeclared_var = True

        return found_use_of_undeclared_var

    def expr_uses_undeclared_var(self, expr, declared, func_name):
        if isinstance(expr, ASTnodes.Number) or isinstance(expr, ASTnodes.Boolean):
            return False
        if isinstance(expr, ASTnodes.Identifier):
            if not expr.name in declared:
                print("ERROR in line %d: Use of undeclared variable '%s'." % (expr.lineno, expr.name))
                return True
            return False
        if isinstance(expr, ASTnodes.Uminus) or isinstance(expr, ASTnodes.Not) or isinstance(expr, ASTnodes.LeakExpr):
            return self.expr_uses_undeclared_var(expr.sub_expr, declared, func_name)
        if isinstance(expr, ASTnodes.Binop):
            return self.expr_uses_undeclared_var(expr.left, declared, func_name) or self.expr_uses_undeclared_var(expr.right, declared, func_name)
        if isinstance(expr, ASTnodes.IfExpr):
            return self.expr_uses_undeclared_var(expr.cond, declared, func_name) or\
                   self.expr_uses_undeclared_var(expr.then_branch, declared, func_name) or\
                   self.expr_uses_undeclared_var(expr.else_branch, declared, func_name)
        if isinstance(expr, ASTnodes.FuncCallExpr):
            undeclared_use = False
            for arg in expr.args:
                undeclared_use = self.expr_uses_undeclared_var(arg, declared, func_name)
            return undeclared_use
        print("Unexpected expression encountered during undeclared var check")
        return True

    def check_io_names(self):
        has_distinct_output_names = True
        has_distinct_input_names = True
        main = self.func_dict['main']
        output_names = set()
        input_names = set()
        for stm in main.body.stms:
            if isinstance(stm, ASTnodes.OutputStm):
                if stm.result_name.name in output_names:
                    has_distinct_output_names = False
                    print("ERROR in line %d: Output name '%s' has already been used previously." % (stm.result_name.lineno,
                                                                                                    stm.result_name.name))
                else:
                    output_names.add(stm.result_name.name)
            elif isinstance(stm, ASTnodes.InputStm):
                if stm.var.name in input_names:
                    has_distinct_input_names = False
                    print("ERROR in line %d: Input name '%s' has already been used previously." % (stm.var.lineno,
                                                                                                   stm.var.name))
                else:
                    input_names.add(stm.var.name)
        return has_distinct_output_names and has_distinct_input_names

    def type_check_bool_num(self, annotating_for_xor = False):
        main = self.func_dict['main']
        types = {}
        for stm in main.body.stms:
            is_sound_stm = self.stm_type_check(stm, types, annotating_for_xor)
            if not is_sound_stm:
                return False
        return True

    def stm_type_check(self, stm, types, annotating_for_xor):
        if isinstance(stm, ASTnodes.ProgramPoint) or \
                isinstance(stm, ASTnodes.JumpIfFalseStm) or \
                isinstance(stm, ASTnodes.JumpUnconditionalStm):
            return
        if isinstance(stm, ASTnodes.OutputStm):
            return True
        if isinstance(stm, ASTnodes.InputStm):
            # Update type map
            types[stm.var.name] = stm.type
            return True
        if not isinstance(stm, ASTnodes.AssignStm):
            print("Unexpected statement encountered during type checking")
            raise SyntaxError
        is_sound_expr = self.expr_type_check(stm.expr, types, annotating_for_xor)
        if not is_sound_expr:
            return False
        # Else, update type map
        types[stm.var.name] = stm.expr.type
        return True

    def expr_type_check(self, expr, types, annotating_for_xor):
        if isinstance(expr, ASTnodes.Number):
            expr.type = 'num'
            return True
        if isinstance(expr, ASTnodes.Boolean):
            expr.type = 'bool'
            return True
        if isinstance(expr, ASTnodes.Identifier) or isinstance(expr, ASTnodes.IfResultId):
            if expr.name == '_randomnum':
                expr.type = 'num'
            elif expr.name == '_randombit':
                expr.type = 'bool'
            else:
                expr.type = types[expr.name]
            return True
        if isinstance(expr, ASTnodes.Uminus) or \
                isinstance(expr, ASTnodes.Not) or \
                isinstance(expr, ASTnodes.LeakExpr):
            sub_expr_sound = self.expr_type_check(expr.sub_expr, types, annotating_for_xor)
            if not sub_expr_sound:
                return False
            if isinstance(expr, ASTnodes.Uminus) and expr.sub_expr.type != 'num' and not annotating_for_xor:
                print("ERROR in line %d: Uminus expression '%s' "
                      "should have subexpression of type NUMBER." % (expr.lineno,
                                                                     expr.readable_str()))
                return False
            if isinstance(expr, ASTnodes.Not) and expr.sub_expr.type != 'bool' and not annotating_for_xor:
                print("ERROR in line %d: Not-expression '%s' "
                      "should have subexpression of type BOOLEAN." % (expr.lineno,
                                                                      expr.readable_str()))
                return False
            expr.type = expr.sub_expr.type
            return True
        if isinstance(expr, ASTnodes.Binop):
            if (not self.expr_type_check(expr.left, types, annotating_for_xor)) or \
                    (not self.expr_type_check(expr.right, types, annotating_for_xor)):
                return False
            if expr.left.type != expr.right.type and not annotating_for_xor:
                print("ERROR in line %d: Operands of binop '%s' have different types." % (expr.lineno,
                                                                                          expr.readable_str()))
                return False
            if expr.left.type == 'num' and not annotating_for_xor and (expr.op == '||' or
                                                                       expr.op == '&&'):
                print("ERROR in line %d: Expression '%s' requires operands of type BOOLEAN." % (expr.lineno,
                                                                                                expr.readable_str()))
                return False
            if expr.left.type == 'bool' and not annotating_for_xor and (expr.op == '+' or
                                                                        expr.op == '-' or
                                                                        expr.op == '*' or
                                                                        expr.op == '/' or
                                                                        expr.op == '<' or
                                                                        expr.op == '>' or
                                                                        expr.op == '<=' or
                                                                        expr.op == '>='):
                print("ERROR in line %d: Expression '%s' requires operands of type NUMBER." % (expr.lineno,
                                                                                               expr.readable_str()))
                return False
            if expr.op == '+' or expr.op == '-' or expr.op == '*' or expr.op == '/':
                expr.type = 'num'
            else:
                expr.type = 'bool'
            return True
        if isinstance(expr, ASTnodes.IfExpr):
            if not self.expr_type_check(expr.cond, types, annotating_for_xor):
                return False
            if expr.cond.type != 'bool' and not annotating_for_xor:
                print("ERROR in line %d: If-condition '%s' must be of type BOOLEAN." % (expr.lineno,
                                                                                        expr.readable_str()))
                return False
            if (not self.expr_type_check(expr.then_branch, types, annotating_for_xor)) or \
                    (not self.expr_type_check(expr.else_branch, types, annotating_for_xor)):
                return False
            if expr.then_branch.type != expr.else_branch.type and not annotating_for_xor:
                print("ERROR in line %d: Then- and else-branch of If-expression must have same types." % expr.lineno)
                return False
            expr.type = expr.then_branch.type
            return True
        if isinstance(expr, ASTnodes.FuncCallExpr):
            if expr.func_id.name == 'randomnum':
                expr.type = 'num'
                return True
            if expr.func_id.name == 'randombit':
                expr.type = 'bool'
                return True
            for arg in expr.args:   # Type check all arguments
                if not self.expr_type_check(arg, types, annotating_for_xor):
                    return False
            callee_name = expr.func_id.name
            callee = deepcopy(self.func_dict[callee_name])
            callee_types = {}
            for callee_arg, caller_arg in zip(callee.args, expr.args):  # Inject types to callee-args
                callee_types[callee_arg.name] = caller_arg.type
            for stm in callee.body.stms:    # Type check statements in function call
                if not self.stm_type_check(stm, callee_types, annotating_for_xor):
                    return False
            if not self.expr_type_check(callee.body.expr, callee_types, annotating_for_xor):    # Type check returned expression
                return False
            expr.type = callee.body.expr.type
            return True
        print("Unexpected expression encountered during type checking.")
        raise SyntaxError

    def rewrite_uminus(self):
        for func in self.prog.funcs:
            for stm in func.body.stms:
                if not isinstance(stm, ASTnodes.AssignStm):
                    continue
                stm.expr = self.expr_remove_uminus(stm.expr)
            if func.id.name != 'main':
                func.body.expr = self.expr_remove_uminus(func.body.expr)

    def expr_remove_uminus(self, expr):
        if isinstance(expr, ASTnodes.Number) or isinstance(expr, ASTnodes.Boolean) or isinstance(expr, ASTnodes.Identifier):
            return expr

        if isinstance(expr, ASTnodes.Not) or isinstance(expr, ASTnodes.LeakExpr):
            expr.sub_expr = self.expr_remove_uminus(expr.sub_expr)
            return expr

        if isinstance(expr, ASTnodes.Binop):
            # Remove uminus in sub-expressions
            expr.left = self.expr_remove_uminus(expr.left)
            expr.right = self.expr_remove_uminus(expr.right)
            return expr

        if isinstance(expr, ASTnodes.IfExpr):
            # Remove uminus in sub-expressions
            expr.cond = self.expr_remove_uminus(expr.cond)
            expr.then_branch = self.expr_remove_uminus(expr.then_branch)
            expr.else_branch = self.expr_remove_uminus(expr.else_branch)
            return expr

        if isinstance(expr, ASTnodes.FuncCallExpr):
            for arg in expr.args:
                arg = self.expr_remove_uminus(arg)
            return expr

        if not isinstance(expr, ASTnodes.Uminus):
            print("Found unexpected Token %s during Uminus-transform" % type(expr))
            raise SyntaxError

        # Else, we actually have a uminus-expression
        return ASTnodes.Binop('-', ASTnodes.Number(0), self.expr_remove_uminus(expr.sub_expr))

    def rewrite_eq(self):
        for func in self.prog.funcs:
            for stm in func.body.stms:
                if not isinstance(stm, ASTnodes.AssignStm):
                    continue
                stm.expr = self.expr_rewrite_eq(stm.expr)
            if func.id.name != 'main':
                func.body.expr = self.expr_rewrite_eq(func.body.expr)

    def expr_rewrite_eq(self, expr):
        if isinstance(expr, ASTnodes.Number) or isinstance(expr, ASTnodes.Boolean) or isinstance(expr, ASTnodes.Identifier):
            return expr
        if isinstance(expr, ASTnodes.Not) or isinstance(expr, ASTnodes.LeakExpr):
            expr.sub_expr = self.expr_rewrite_eq(expr.sub_expr)
            return expr
        if isinstance(expr, ASTnodes.FuncCallExpr):
            for arg in expr.args:
                arg = self.expr_rewrite_eq(arg)
            return expr
        if isinstance(expr, ASTnodes.IfExpr):
            expr.cond = self.expr_rewrite_eq(expr.cond)
            expr.then_branch = self.expr_rewrite_eq(expr.then_branch)
            expr.else_branch = self.expr_rewrite_eq(expr.else_branch)
            return expr
        if isinstance(expr, ASTnodes.Binop):
            expr.left = self.expr_rewrite_eq(expr.left)
            expr.right = self.expr_rewrite_eq(expr.right)
            if expr.op == '==':
                return ASTnodes.Not(ASTnodes.Binop(op='!=',
                                                   left=expr.left,
                                                   right=expr.right))
            return expr
        print("Unexpected expression encountered during NEQ rewriting.")
        raise SyntaxError

    def change_reused_var_names(self):
        for func in self.prog.funcs:
            var_names = {}
            if func.id.name == 'main':  # This prevent var names of input stms from being changed.
                for stm in func.body.stms:
                    if isinstance(stm, ASTnodes.InputStm):
                        var_names[stm.var.name] = stm.var.name
            for arg in func.args:
                var_names[arg.name] = arg.name
            for stm in func.body.stms:
                if isinstance(stm, ASTnodes.InputStm):
                    var_names[stm.var.name] = stm.var.name  # Update current name of variable
                elif isinstance(stm, ASTnodes.OutputStm):
                    self.expr_change_reused_var_names(stm.output_var, var_names)
                elif isinstance(stm, ASTnodes.AssignStm):
                    self.expr_change_reused_var_names(stm.expr, var_names)
                    if stm.var.name in var_names:
                        new_var_name = "_%s_%d" % (stm.var.name, self.var_redef_counter)
                        self.var_redef_counter += 1
                        var_names[stm.var.name] = new_var_name
                        stm.var.name = new_var_name
                    else:
                        var_names[stm.var.name] = stm.var.name
                else:
                    print("Unexpected statement encountered during reused var renaming")
                    raise SyntaxError
            if func.id.name != 'main':
                self.expr_change_reused_var_names(func.body.expr, var_names)

    def expr_change_reused_var_names(self, expr, var_names):
        if isinstance(expr, ASTnodes.Identifier):
            if expr.name in var_names:
                expr.name = var_names[expr.name]
        elif isinstance(expr, ASTnodes.Not) or isinstance(expr, ASTnodes.LeakExpr):
            self.expr_change_reused_var_names(expr.sub_expr, var_names)
        elif isinstance(expr, ASTnodes.Binop):
            self.expr_change_reused_var_names(expr.left, var_names)
            self.expr_change_reused_var_names(expr.right, var_names)
        elif isinstance(expr, ASTnodes.IfExpr):
            self.expr_change_reused_var_names(expr.cond, var_names)
            self.expr_change_reused_var_names(expr.then_branch, var_names)
            self.expr_change_reused_var_names(expr.else_branch, var_names)
        elif isinstance(expr, ASTnodes.FuncCallExpr):
            for arg in expr.args:
                self.expr_change_reused_var_names(arg, var_names)

    def smart_inline_program(self):
        body = self.func_smart_inline('main', {})
        main = ASTnodes.Function(id=ASTnodes.Identifier('main'),
                                 args=[],
                                 body=ASTnodes.FunctionBody(body.stms, body.expr))
        self.prog.funcs = [main]
        self.func_dict = {'main': main}
        self.rewrite_ifs_and_remove_body_exprs()

    def func_smart_inline(self, func_name, var_values):
        # var_values contains expressions for all (renamed) args
        func = deepcopy(self.func_dict[func_name])
        if func.id.name != 'main':
            self.rename_all_vars(func)

        arg_assignments = []
        for k, v in var_values.items():
            assignment = ASTnodes.AssignStm(var=ASTnodes.Identifier(k),
                                            expr=v)
            arg_assignments.append(assignment)

        for stm in func.body.stms:
            if isinstance(stm, ASTnodes.InputStm):  # Relevant in main
                var_values[stm.var.name] = ASTnodes.Identifier(name=stm.var.name,
                                                               is_public=False)
            elif isinstance(stm, ASTnodes.AssignStm):
                stm.expr = self.expr_smart_inline(stm.expr, var_values)
                var_values[stm.var.name] = stm.expr

        func.body.stms = arg_assignments + func.body.stms
        if func_name != 'main':
            func.body.expr = self.expr_smart_inline(func.body.expr, var_values)
        return func.body

    def expr_smart_inline(self, expr, var_values):
        if isinstance(expr, ASTnodes.Number) or isinstance(expr, ASTnodes.Boolean):
            expr.is_public = True
            return expr
        if isinstance(expr, ASTnodes.Identifier):
            expr.is_public = var_values[expr.name].is_public_exp()
            return expr
        if isinstance(expr, ASTnodes.LeakExpr):
            expr.sub_expr = self.expr_smart_inline(expr.sub_expr, var_values)
            expr.is_public = True   # This might already have been done when creating AST node...
            if self.private_cond_depth > 0:
                print("WARNING in line %d: Leaking in secret branch may leak value of branch-condition." % expr.lineno)
            return expr             # Could also set sub_expr to be public...
        if isinstance(expr, ASTnodes.Not):
            expr.sub_expr = self.expr_smart_inline(expr.sub_expr, var_values)
            expr.is_public = expr.sub_expr.is_public_exp()
            return expr
        if isinstance(expr, ASTnodes.Binop):
            expr.left = self.expr_smart_inline(expr.left, var_values)
            expr.right = self.expr_smart_inline(expr.right, var_values)
            expr.is_public = expr.left.is_public_exp() and expr.right.is_public_exp()
            return expr
        if isinstance(expr, ASTnodes.IfExpr):
            expr.cond = self.expr_smart_inline(expr.cond, var_values)
            expr.is_public = expr.cond.is_public_exp()
            if not expr.is_public_exp():
                self.private_cond_depth += 1
            expr.then_branch = self.expr_smart_inline(expr.then_branch, var_values)
            expr.else_branch = self.expr_smart_inline(expr.else_branch, var_values)
            self.private_cond_depth -= 1
            return expr
        if isinstance(expr, ASTnodes.FuncCallExpr):
            callee_name = expr.func_id.name
            if callee_name == 'randomnum' or callee_name == 'randombit':
                return ASTnodes.Identifier('_%s' % callee_name)
            for arg in expr.args:
                arg = self.expr_smart_inline(arg, var_values)
            callee = self.func_dict[callee_name]
            arg_values = {}
            for callee_arg, caller_arg in zip(callee.args, expr.args):
                new_callee_arg = deepcopy(callee_arg)
                self.expr_rename_vars(new_callee_arg, callee)
                arg_values[new_callee_arg.name] = caller_arg
            return self.func_smart_inline(callee_name, arg_values)

        print("Unexpected expression encountered during smart inline")
        raise SyntaxError

    def rename_all_vars(self, func):
        for stm in func.body.stms:
            if isinstance(stm, ASTnodes.InputStm):
                self.expr_rename_vars(stm.var, func)
            elif isinstance(stm, ASTnodes.OutputStm):
                self.expr_rename_vars(stm.output_var, func)
            elif isinstance(stm, ASTnodes.AssignStm):
                self.expr_rename_vars(stm.var, func)
                self.expr_rename_vars(stm.expr, func)
            else:
                print("Unexpected statement found during function renaming.")
                raise SyntaxError

        if func.id.name != 'main':
            self.expr_rename_vars(func.body.expr, func)
        self.func_call_counter += 1

    def expr_rename_vars(self, expr, func):
        if isinstance(expr, ASTnodes.Identifier):
            expr.name = '_%s_%d_%s' % (func.id.name, self.func_call_counter, expr.name)
        elif isinstance(expr, ASTnodes.Not) or isinstance(expr, ASTnodes.LeakExpr):
            self.expr_rename_vars(expr.sub_expr, func)
        elif isinstance(expr, ASTnodes.Binop):
            self.expr_rename_vars(expr.left, func)
            self.expr_rename_vars(expr.right, func)
        elif isinstance(expr, ASTnodes.FuncCallExpr):
            for arg in expr.args:
                self.expr_rename_vars(arg, func)
        elif isinstance(expr, ASTnodes.IfExpr):
            self.expr_rename_vars(expr.cond, func)
            self.expr_rename_vars(expr.then_branch, func)
            self.expr_rename_vars(expr.else_branch, func)

    def rewrite_ifs_and_remove_body_exprs(self):
        main = self.func_dict['main']
        new_stms = []
        for stm in main.body.stms:
            self.stm_if_and_body_rewrite(stm, new_stms)
        main.body.stms = new_stms

    def stm_if_and_body_rewrite(self, stm, stms):
        if not isinstance(stm, ASTnodes.AssignStm):
            stms.append(stm)
            return
        result_expr = self.expr_if_and_body_rewrite(stm.expr, stms)
        stms.append(ASTnodes.AssignStm(stm.var, result_expr))

    def expr_if_and_body_rewrite(self, expr, stms):
        if isinstance(expr, ASTnodes.Number) or isinstance(expr, ASTnodes.Boolean) or isinstance(expr, ASTnodes.Identifier):
            return expr
        if isinstance(expr, ASTnodes.Not) or isinstance(expr, ASTnodes.LeakExpr):
            expr.sub_expr = self.expr_if_and_body_rewrite(expr.sub_expr, stms)
            return expr
        if isinstance(expr, ASTnodes.Binop):
            expr.left = self.expr_if_and_body_rewrite(expr.left, stms)
            expr.right = self.expr_if_and_body_rewrite(expr.right, stms)
            return expr
        if isinstance(expr, ASTnodes.FunctionBody):
            for stm in expr.stms:
                self.stm_if_and_body_rewrite(stm, stms)
            return self.expr_if_and_body_rewrite(expr.expr, stms)
        if isinstance(expr, ASTnodes.IfExpr):
            is_public = expr.cond.is_public_exp()
            # First compute the condition, and save it in tmp variable
            expr.cond = self.expr_if_and_body_rewrite(expr.cond, stms)
            cond_var = '_tmp%s' % str(self.tmp_counter)
            self.tmp_counter += 1
            cond_assign = ASTnodes.AssignStm(var=ASTnodes.Identifier(cond_var),
                                             expr=expr.cond)
            stms.append(cond_assign)

            if not is_public:
                # Compute both branches to hide which one is chosen
                expr.then_branch = self.expr_if_and_body_rewrite(expr.then_branch, stms)
                expr.else_branch = self.expr_if_and_body_rewrite(expr.else_branch, stms)
                # Make multiplexer to get correct value
                left_bool = ASTnodes.Identifier(cond_var)
                left = ASTnodes.Binop('*', left_bool, expr.then_branch)

                right_bool = ASTnodes.Binop('-', ASTnodes.Number(1), ASTnodes.Identifier(cond_var))
                right = ASTnodes.Binop('*', right_bool, expr.else_branch)
                return ASTnodes.Binop('+', left, right)
            # Else, IfExpr is public, and we only need to compute the correct branch
            result_var = '_tmp%s' % str(self.tmp_counter)
            self.tmp_counter += 1

            jz_stm = ASTnodes.JumpIfFalseStm(var=ASTnodes.Identifier(cond_var),
                                             destination=ASTnodes.ProgramPoint(self.program_point_counter))
            jmp_stm = ASTnodes.JumpUnconditionalStm(destination=ASTnodes.ProgramPoint(self.program_point_counter + 1))
            point_before_else = ASTnodes.ProgramPoint(self.program_point_counter)
            exit_point = ASTnodes.ProgramPoint(self.program_point_counter + 1)
            self.program_point_counter += 2

            stms.append(jz_stm)
            then_result = self.expr_if_and_body_rewrite(expr.then_branch, stms)
            stms.append(ASTnodes.AssignStm(var=ASTnodes.Identifier(result_var),
                                           expr=then_result,
                                           is_if_result_assign=True))
            stms.append(jmp_stm)
            stms.append(point_before_else)
            else_result = self.expr_if_and_body_rewrite(expr.else_branch, stms)
            stms.append(ASTnodes.AssignStm(var=ASTnodes.Identifier(result_var),
                                           expr=else_result,
                                           is_if_result_assign=True))
            stms.append(exit_point)
            return ASTnodes.IfResultId(result_var)

        print("Unexpected expression encountered during if and body rewrite")
        raise SyntaxError

    def insert_tmps(self):
        main = self.func_dict['main']
        new_stms = []
        for stm in main.body.stms:
            if not isinstance(stm, ASTnodes.AssignStm):
                # No expressions to fold out with tmp variables
                new_stms.append(stm)
                continue
            # Else, we have an AssignStm, and must first fold out right-hand side expression
            if not (isinstance(stm.expr, ASTnodes.Binop) or
                    isinstance(stm.expr, ASTnodes.Not) or
                    isinstance(stm.expr, ASTnodes.LeakExpr)):
                # No sub-expressions to create tmps for. We eliminated uminus, ifs, func calls..
                new_stms.append(stm)
                continue
            if isinstance(stm.expr, ASTnodes.Binop):
                # We have a Binop. Fold out subexpressions
                left = self.expr_insert_tmps(stm.expr.left, new_stms)
                right = self.expr_insert_tmps(stm.expr.right, new_stms)
                # Now, both sides of binop have only number/identifiers
                stm.expr = ASTnodes.Binop(stm.expr.op, left, right)  # Is this even necessary?
            elif isinstance(stm.expr, ASTnodes.Not):
                sub_expr = self.expr_insert_tmps(stm.expr.sub_expr, new_stms)
                stm.expr = ASTnodes.Not(sub_expr)
            elif isinstance(stm.expr, ASTnodes.LeakExpr):
                sub_expr = self.expr_insert_tmps(stm.expr.sub_expr, new_stms)
                stm.expr = ASTnodes.LeakExpr(sub_expr)
            # Finally, add the statement to the list
            new_stms.append(stm)
        main.body.stms = new_stms

    def expr_insert_tmps(self, expr, stms):
        if isinstance(expr, ASTnodes.Number) or isinstance(expr, ASTnodes.Boolean):
            return expr
        if isinstance(expr, ASTnodes.Identifier) or isinstance(expr, ASTnodes.IfResultId):
            if expr.name == '_randomnum' or expr.name == '_randombit':
                tmp_var = 'tmp%s' % str(self.tmp_counter)
                self.tmp_counter += 1
                assignment = ASTnodes.AssignStm(ASTnodes.Identifier(tmp_var), expr)
                stms.append(assignment)
                return ASTnodes.Identifier(tmp_var)
            return expr

        if not (isinstance(expr, ASTnodes.LeakExpr) or
                isinstance(expr, ASTnodes.Not) or
                isinstance(expr, ASTnodes.Binop)):
            print("Unexpected expression encountered during tmp insertion")
            raise SyntaxError

        tmp_var = '_tmp%s' % str(self.tmp_counter)
        self.tmp_counter += 1

        if isinstance(expr, ASTnodes.LeakExpr):
            sub_expr = self.expr_insert_tmps(expr.sub_expr, stms)
            new_expr = ASTnodes.LeakExpr(sub_expr)
        elif isinstance(expr, ASTnodes.Not):
            sub_expr = self.expr_insert_tmps(expr.sub_expr, stms)
            new_expr = ASTnodes.Not(sub_expr)
        elif isinstance(expr, ASTnodes.Binop):
            # Else, we have a binop. First insert tmps for sub-expressions
            left = self.expr_insert_tmps(expr.left, stms)
            right = self.expr_insert_tmps(expr.right, stms)
            new_expr = ASTnodes.Binop(expr.op, left, right)

        assignment = ASTnodes.AssignStm(ASTnodes.Identifier(tmp_var), new_expr)
        stms.append(assignment)

        return ASTnodes.Identifier(tmp_var)

    def constant_propagation(self):
        values = {}
        values['_randomnum'] = Value(is_constant=False)
        values['_randombit'] = Value(is_constant=False)
        stms = []
        main = self.func_dict['main']
        for stm in main.body.stms:
            if isinstance(stm, ASTnodes.ProgramPoint) or isinstance(stm, ASTnodes.JumpUnconditionalStm):
                stms.append(stm)
                continue
            if isinstance(stm, ASTnodes.InputStm):
                values[stm.var.name] = Value(is_constant=False)
                stms.append(stm)
                continue
            if isinstance(stm, ASTnodes.OutputStm):
                value = values[stm.output_var.name]
                if value.is_constant:
                    print("WARNING: Outputting variable '%s' with constant value %s" % (stm.output_var.name, value.value))
                    if value.value is False or value.value is True:
                        stm.output_var = ASTnodes.Boolean(value.value)
                    else:
                        stm.output_var = ASTnodes.Number(value.value)
                stms.append(stm)
                continue
            if isinstance(stm, ASTnodes.JumpIfFalseStm):
                value = values[stm.var.name]
                if not value.is_constant:
                    stms.append(stm)
                elif not value.value:   # If constant value is False, we jump unconditionally
                    stms.append(ASTnodes.JumpUnconditionalStm(stm.destination))
                continue    # Otherwise we should NOT jump, and just do not append anything
            if not isinstance(stm, ASTnodes.AssignStm):
                print("Unexpected statement encountered during constant propagation.")
                raise SyntaxError
            # Else, it is an AssignStm
            if stm.is_if_result_assign:
                stm.expr = self.eval_expr(stm.expr, values)
                stms.append(stm)    # Necessary to do this statement
                values[stm.var.name] = Value(is_constant=False)
                continue
            if isinstance(stm.expr, ASTnodes.Number) or isinstance(stm.expr, ASTnodes.Boolean):
                values[stm.var.name] = Value(True, stm.expr.value)
                continue  # No need to do this assignment
            if isinstance(stm.expr, ASTnodes.Identifier):
                if stm.expr.name == '_randomnum' or stm.expr.name == '_randombit':
                    values[stm.var.name] = Value(is_constant=False)
                    stms.append(stm)
                    continue
                # Else
                self.rename_occurrences_of_var(old_name=stm.var.name,
                                               new_name=stm.expr.name)
                continue
            if isinstance(stm.expr, ASTnodes.IfResultId):
                self.rename_occurrences_of_var(old_name=stm.var.name,
                                               new_name=stm.expr.name)
                # We do not make the assignment, since we renamed all occurrences of the variable.
                continue
            if isinstance(stm.expr, ASTnodes.LeakExpr):
                if isinstance(stm.expr.sub_expr, ASTnodes.Number) or isinstance(stm.expr.sub_expr, ASTnodes.Boolean):
                    values[stm.var.name] = Value(True, stm.expr.sub_expr.value)
                    # No need to do assignment or leak in this case
                elif isinstance(stm.expr.sub_expr, ASTnodes.Identifier):
                    values[stm.var.name] = deepcopy(values[stm.expr.sub_expr.name])
                    if not values[stm.var.name].is_constant:
                        stms.append(stm)
                else:
                    print("Unexpected subexpression in LeakExpr encountered during constant propagation.")
                    raise SyntaxError
                continue
            if isinstance(stm.expr, ASTnodes.Not):
                if isinstance(stm.expr.sub_expr, ASTnodes.Boolean):
                    values[stm.var.name] = Value(is_constant=True,
                                                 value=not stm.expr.sub_expr.value)
                    # No need to do assignment in this case
                elif isinstance(stm.expr.sub_expr, ASTnodes.Identifier):
                    value = deepcopy(values[stm.expr.sub_expr.name])
                    if not value.is_constant:
                        stms.append(stm)
                    else:   # Do the actual negation
                        value.value = not value.value
                    values[stm.var.name] = value
                else:
                    print("Unexpected subexpression in Not-expression encountered during constant propagation.")
                    raise SyntaxError
                continue
            if isinstance(stm.expr, ASTnodes.Binop):
                if isinstance(stm.expr.left, ASTnodes.Number) or isinstance(stm.expr.left, ASTnodes.Boolean):
                    left_value = Value(True, stm.expr.left.value)
                else:
                    left_value = values[stm.expr.left.name]

                if isinstance(stm.expr.right, ASTnodes.Number) or isinstance(stm.expr.right, ASTnodes.Boolean):
                    right_value = Value(True, stm.expr.right.value)
                else:
                    right_value = values[stm.expr.right.name]

                op = stm.expr.op
                if left_value.is_constant and right_value.is_constant:
                    if op == '+':
                        values[stm.var.name] = Value(True, left_value.value + right_value.value)
                    elif op == '-':
                        values[stm.var.name] = Value(True, left_value.value - right_value.value)
                    elif op == '*':
                        values[stm.var.name] = Value(True, left_value.value * right_value.value)
                    elif op == '/':
                        values[stm.var.name] = Value(True, left_value.value // right_value.value)
                    elif op == '||':
                        values[stm.var.name] = Value(True, left_value.value or right_value.value)
                    elif op == '&&':
                        values[stm.var.name] = Value(True, left_value.value and right_value.value)
                    elif op == '==':
                        values[stm.var.name] = Value(True, left_value.value == right_value.value)
                    elif op == '!=':
                        values[stm.var.name] = Value(True, left_value.value != right_value.value)
                    elif op == '<':
                        values[stm.var.name] = Value(True, left_value.value < right_value.value)
                    elif op == '>':
                        values[stm.var.name] = Value(True, left_value.value > right_value.value)
                    elif op == '<=':
                        values[stm.var.name] = Value(True, left_value.value <= right_value.value)
                    elif op == '>=':
                        values[stm.var.name] = Value(True, left_value.value >= right_value.value)
                    else:
                        print("Unexpected operator encountered during constant propagation")
                        raise SyntaxError
                    # No need to have assignment
                elif (left_value.is_constant or right_value.is_constant) and (op == '||' or
                                                                              op == '&&' or
                                                                              op == '==' or
                                                                              op == '!='):
                    # Smart boolean propagation
                    if right_value.is_constant:     # Swap, so we always have constant on left side
                        left_value, right_value = right_value, left_value
                        stm.expr.left, stm.expr.right = stm.expr.right, stm.expr.left
                    if not (left_value.value is False or left_value.value is True):
                        values[stm.var.name] = Value(is_constant=False)
                        if left_value.is_constant:
                            if left_value.value is False or left_value.value is True:
                                stm.expr.left = ASTnodes.Boolean(left_value.value)
                            else:
                                stm.expr.left = ASTnodes.Number(left_value.value)
                        if right_value.is_constant:
                            if right_value.value is False or right_value.value is True:
                                stm.expr.right = ASTnodes.Boolean(right_value.value)
                            else:
                                stm.expr.right = ASTnodes.Number(right_value.value)
                        stms.append(stm)
                        continue
                    if op == '||':
                        if left_value.value is True:
                            values[stm.var.name] = Value(is_constant=True, value=True)
                        else:
                            values[stm.var.name] = Value(is_constant=False)
                            stm.expr = stm.expr.right
                            stms.append(stm)
                    elif op == '&&':
                        if left_value.value is False:
                            values[stm.var.name] = Value(is_constant=True, value=False)
                        else:
                            values[stm.var.name] = Value(is_constant=False)
                            stm.expr = stm.expr.right
                            stms.append(stm)
                    elif op == '==':
                        if left_value.value is True:
                            values[stm.var.name] = Value(is_constant=False)
                            stm.expr = stm.expr.right
                            stms.append(stm)
                        else:
                            values[stm.var.name] = Value(is_constant=False)
                            stm.expr = ASTnodes.Not(stm.expr.right)
                            stms.append(stm)
                    elif op == '!=':
                        if left_value.value is False:
                            values[stm.var.name] = Value(is_constant=False)
                            stm.expr = stm.expr.right
                            stms.append(stm)
                        else:
                            values[stm.var.name] = Value(is_constant=False)
                            stm.expr = ASTnodes.Not(stm.expr.right)
                            stms.append(stm)
                else:
                    values[stm.var.name] = Value(is_constant=False)

                    if left_value.is_constant:
                        if left_value.value is False or left_value.value is True:
                            stm.expr.left = ASTnodes.Boolean(left_value.value)
                        else:
                            stm.expr.left = ASTnodes.Number(left_value.value)
                    if right_value.is_constant:
                        if right_value.value is False or right_value.value is True:
                            stm.expr.right = ASTnodes.Boolean(right_value.value)
                        else:
                            stm.expr.right = ASTnodes.Number(right_value.value)
                    stms.append(stm)
        main.body.stms = stms

    def eval_expr(self, expr, values):
        if isinstance(expr, ASTnodes.Number) or isinstance(expr, ASTnodes.Boolean):
            return expr
        if isinstance(expr, ASTnodes.Identifier):
            value = values[expr.name]
            if value.is_constant:
                if value.value is False or value.value is True:
                    return ASTnodes.Boolean(value.value)
                return ASTnodes.Number(value.value)
            return expr
        if isinstance(expr, ASTnodes.IfResultId):
            return ASTnodes.Identifier(expr.name)   # TODO: Not too sure about this...
        if isinstance(expr, ASTnodes.LeakExpr):
            return self.eval_expr(expr.sub_expr, values)
        if isinstance(expr, ASTnodes.Not):
            if isinstance(expr.sub_expr, ASTnodes.Boolean):
                sub_expr_value = Value(True, expr.sub_expr.value)
            else:
                sub_expr_value = values[expr.sub_expr.name]

            if sub_expr_value.is_constant:
                return ASTnodes.Boolean(not sub_expr_value.value)
            return expr
        if isinstance(expr, ASTnodes.Binop):
            if isinstance(expr.left, ASTnodes.Number) or isinstance(expr.left, ASTnodes.Boolean):
                left_value = Value(True, expr.left.value)
            else:
                left_value = values[expr.left.name]

            if isinstance(expr.right, ASTnodes.Number) or isinstance(expr.right, ASTnodes.Boolean):
                right_value = Value(True, expr.right.value)
            else:
                right_value = values[expr.right.name]

            if left_value.is_constant and right_value.is_constant:
                op = expr.op
                if op == '+':
                    return ASTnodes.Number(left_value.value + right_value.value)
                elif op == '-':
                    return ASTnodes.Number(left_value.value - right_value.value)
                elif op == '*':
                    return ASTnodes.Number(left_value.value * right_value.value)
                elif op == '/':
                    return ASTnodes.Number(left_value.value // right_value.value)
                elif op == '||':
                    return ASTnodes.Boolean(left_value.value or right_value.value)
                elif op == '&&':
                    return ASTnodes.Boolean(left_value.value and right_value.value)
                elif op == '==':
                    return ASTnodes.Boolean(left_value.value == right_value.value)
                elif op == '!=':
                    return ASTnodes.Boolean(left_value.value != right_value.value)
                elif op == '<':
                    return ASTnodes.Boolean(left_value.value < right_value.value)
                elif op == '>':
                    return ASTnodes.Boolean(left_value.value > right_value.value)
                elif op == '<=':
                    return ASTnodes.Boolean(left_value.value <= right_value.value)
                elif op == '>=':
                    return ASTnodes.Boolean(left_value.value >= right_value.value)
                else:
                    print("Unexpected operator encountered during constant propagation")
                    raise SyntaxError
            return expr

    def rename_occurrences_of_var(self, old_name, new_name):
        main = self.func_dict['main']
        for stm in main.body.stms:
            if isinstance(stm, ASTnodes.OutputStm) and stm.output_var.name == old_name:
                stm.output_var.name = new_name
            elif isinstance(stm, ASTnodes.JumpIfFalseStm) and stm.var.name == old_name:
                stm.var.name = new_name
            elif isinstance(stm, ASTnodes.AssignStm):
                self.expr_rename_occurrences_of_var(stm.expr, old_name, new_name)

    def expr_rename_occurrences_of_var(self, expr, old_name, new_name):
        if isinstance(expr, ASTnodes.Identifier) and expr.name == old_name:
            expr.name = new_name
        elif isinstance(expr, ASTnodes.IfResultId) and expr.name == old_name:
            expr.name = new_name
        elif isinstance(expr, ASTnodes.Not) or isinstance(expr, ASTnodes.LeakExpr):
            self.expr_rename_occurrences_of_var(expr.sub_expr, old_name, new_name)
        elif isinstance(expr, ASTnodes.Binop):
            self.expr_rename_occurrences_of_var(expr.left, old_name, new_name)
            self.expr_rename_occurrences_of_var(expr.right, old_name, new_name)

    def introduce_xor(self):
        main = self.func_dict['main']
        for stm in main.body.stms:
            if isinstance(stm, ASTnodes.AssignStm) and \
                    isinstance(stm.expr, ASTnodes.Binop) and \
                    stm.expr.left.type == 'bool' and \
                    stm.expr.op == '!=':
                stm.expr.op = 'xor'

    def emit_instructions(self):
        instructions = []
        main = self.func_dict['main']
        for stm in main.body.stms:
            instructions.append(self.translate_instruction(stm))
        return instructions

    def translate_instruction(self, stm):
        if isinstance(stm, ASTnodes.InputStm):
            return "INPUT %d %s" % (stm.input_provider.value, stm.var.name)
        if isinstance(stm, ASTnodes.OutputStm):
            return "OUTPUT %s %s" % (stm.output_var.readable_str(), stm.result_name.readable_str())
        if isinstance(stm, ASTnodes.ProgramPoint):
            return "PROGRAM_POINT %d" % stm.num
        if isinstance(stm, ASTnodes.JumpUnconditionalStm):
            return "JMP %d" % stm.destination.num
        if isinstance(stm, ASTnodes.JumpIfFalseStm):
            return "JZ %s %d" % (stm.var.readable_str(), stm.destination.num)
        if not (isinstance(stm, ASTnodes.AssignStm) and (isinstance(stm.expr, ASTnodes.Binop) or
                                                         isinstance(stm.expr, ASTnodes.Not) or
                                                         isinstance(stm.expr, ASTnodes.Number) or
                                                         isinstance(stm.expr, ASTnodes.Boolean) or
                                                         isinstance(stm.expr, ASTnodes.Identifier) or
                                                         isinstance(stm.expr, ASTnodes.LeakExpr))):
            print("Unexpected statement while translating instructions.")
            return None
        if isinstance(stm.expr, ASTnodes.Number) or \
                isinstance(stm.expr, ASTnodes.Boolean) or \
                isinstance(stm.expr, ASTnodes.Identifier):
            if isinstance(stm.expr, ASTnodes.Identifier) and (stm.expr.name == '_randomnum'):
                return "RANDOM %s" % stm.var.name
            if isinstance(stm.expr, ASTnodes.Identifier) and (stm.expr.name == '_randombit'):
                return "RANDOM_BIT %s" % stm.var.name
            return "MOVE %s %s" % (stm.expr.readable_str(), stm.var.name)
        if isinstance(stm.expr, ASTnodes.LeakExpr):
            return 'LEAK %s %s' % (stm.expr.sub_expr.readable_str(), stm.var.name)
        if isinstance(stm.expr, ASTnodes.Not):
            return 'NOT %s %s' % (stm.expr.sub_expr.readable_str(), stm.var.name)
        # Else, we have a Binop
        op = stm.expr.op
        if op == '+':
            return 'PLUS %s %s %s' % (stm.expr.left.readable_str(), stm.expr.right.readable_str(), stm.var.name)
        if op == '*':
            return 'MULTIPLY %s %s %s' % (stm.expr.left.readable_str(), stm.expr.right.readable_str(), stm.var.name)
        if op == '-':
            return 'MINUS %s %s %s' % (stm.expr.left.readable_str(), stm.expr.right.readable_str(), stm.var.name)
        if op == '/':
            return 'DIVIDE %s %s %s' % (stm.expr.left.readable_str(), stm.expr.right.readable_str(), stm.var.name)
        if op == '||':
            return 'OR %s %s %s' % (stm.expr.left.readable_str(), stm.expr.right.readable_str(), stm.var.name)
        if op == '&&':
            return 'AND %s %s %s' % (stm.expr.left.readable_str(), stm.expr.right.readable_str(), stm.var.name)
        if op == 'xor':
            return 'XOR %s %s %s' % (stm.expr.left.readable_str(), stm.expr.right.readable_str(), stm.var.name)
        if op == '==':
            return 'EQUALS %s %s %s' % (stm.expr.left.readable_str(), stm.expr.right.readable_str(), stm.var.name)
        if op == '!=':
            return 'NOT_EQUALS %s %s %s' % (stm.expr.left.readable_str(), stm.expr.right.readable_str(), stm.var.name)
        if op == '<':
            return 'LT %s %s %s' % (stm.expr.left.readable_str(), stm.expr.right.readable_str(), stm.var.name)
        if op == '>':
            return 'GT %s %s %s' % (stm.expr.left.readable_str(), stm.expr.right.readable_str(), stm.var.name)
        if op == '<=':
            return 'LTE %s %s %s' % (stm.expr.left.readable_str(), stm.expr.right.readable_str(), stm.var.name)
        if op == '>=':
            return 'GTE %s %s %s' % (stm.expr.left.readable_str(), stm.expr.right.readable_str(), stm.var.name)

        print("Operator %s not yet supported for translation :(" % op)
        return None


class Value:
    def __init__(self, is_constant, value=None):
        self.is_constant = is_constant
        self.value = value

    def __str__(self):
        if self.is_constant:
            return 'Value(%s)' % str(self.value)
        return 'Value(secret)'
