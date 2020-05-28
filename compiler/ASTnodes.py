class Program:
    def __init__(self, funcs, prime = None):
        self.funcs = funcs
        self.prime = prime

    def __str__(self):
        return '(PROGRAM, [%s])' % ', '.join([str(elem) for elem in self.funcs])

    def readable_str(self):
        return '\n\n'.join([elem.readable_str() for elem in self.funcs])


class Function:
    def __init__(self, id, args, body, lineno=None):
        self.id = id
        self.args = args
        self.body = body
        self.lineno = lineno

    def __str__(self):
        return '(FUNC, %s, [%s], %s)' % (str(self.id), ', '.join([str(elem) for elem in self.args]), str(self.body))

    def readable_str(self):
        return '%s (%s) { \n\t%s}' % (self.id.readable_str(), ', '.join([elem.readable_str() for elem in self.args]), self.body.readable_str())


class FunctionBody:
    def __init__(self, stms, expr):
        self.stms = stms
        self.expr = expr

    def is_public_exp(self):
        return self.expr.is_public_exp()

    def __str__(self):
        if self.expr is not None:
            return '(BODY, [%s], %s)' % (', '.join([str(elem) for elem in self.stms]), str(self.expr))
        return '(BODY, [%s])' % ', '.join([str(elem) for elem in self.stms])

    def readable_str(self):
        if self.expr is not None:
            return '%s\n\t%s\n' % ('\n\t'.join([elem.readable_str() for elem in self.stms]), self.expr.readable_str())
        return '%s\n' % '\n\t'.join([elem.readable_str() for elem in self.stms])


class Identifier:
    def __init__(self, name, lineno=None, is_public=None):
        self.name = name
        self.lineno = lineno
        self.is_public = is_public

    def is_public_exp(self):
        return self.is_public

    def __str__(self):
        return '(ID, %s)' % self.name

    def readable_str(self):
        return "%s" % self.name


class AssignStm:
    def __init__(self, var, expr, lineno=None, is_if_result_assign = False):
        self.var = var
        self.expr = expr
        self.lineno = lineno
        self.is_if_result_assign = is_if_result_assign

    def __str__(self):
        return '(ASSIGN, %s, %s)' % (str(self.var), str(self.expr))

    def readable_str(self):
        return '%s = %s;' % (self.var.readable_str(), self.expr.readable_str())


class InputStm:
    def __init__(self, var, input_provider, type, lineno=None):
        self.var = var
        self.input_provider = input_provider
        self.type = type
        self.lineno = lineno

    def __str__(self):
        return '(INPUT, %s, %s, %s)' % (str(self.var), str(self.input_provider), self.type)

    def readable_str(self):
        return '%s << %s : %s;' % (self.var.readable_str(), self.input_provider.readable_str(), self.type)


class OutputStm:
    def __init__(self, output_var, result_name, lineno=None):
        self.output_var = output_var
        self.result_name = result_name
        self.lineno = lineno

    def __str__(self):
        return '(OUTPUT, %s, %s)' % (str(self.output_var), str(self.result_name))

    def readable_str(self):
        return '%s >> %s;' % (self.output_var.readable_str(), self.result_name.readable_str())


class Binop:
    def __init__(self, op, left, right, lineno=None, is_public=None):
        self.op = op
        self.left = left
        self.right = right
        self.lineno = lineno
        self.is_public = is_public

    def is_public_exp(self):
        return self.is_public

    def __str__(self):
        return '(BINOP, %s, %s, %s)' % (str(self.op), str(self.left), str(self.right))

    def readable_str(self):
        return '(%s %s %s)' % (self.left.readable_str(), self.op, self.right.readable_str())


class LeakExpr:
    def __init__(self, expr, lineno=None):
        self.sub_expr = expr
        self.lineno = lineno
        self.is_public = True

    def is_public_exp(self):
        return True

    def __str__(self):
        return '(LEAK, %s)' % str(self.sub_expr)

    def readable_str(self):
        return 'leak %s' % self.sub_expr.readable_str()


class Uminus:
    def __init__(self, expr, lineno=None, is_public=None):
        self.sub_expr = expr
        self.lineno = lineno
        self.is_public = is_public

    def is_public_exp(self):
        return self.is_public

    def __str__(self):
        return '(UMINUS, %s)' % str(self.sub_expr)

    def readable_str(self):
        return '(-%s)' % self.sub_expr.readable_str()


class Not:
    def __init__(self, expr, lineno=None, is_public=None):
        self.sub_expr = expr
        self.lineno = lineno
        self.is_public = is_public

    def is_public_exp(self):
        return self.is_public

    def __str__(self):
        return '(NOT, %s)' % str(self.sub_expr)

    def readable_str(self):
        return '!%s' % self.sub_expr.readable_str()


class Number:
    def __init__(self, value, lineno=None, is_public=None):
        self.value = value
        self.lineno = lineno
        self.is_public = is_public

    def is_public_exp(self):
        return self.is_public

    def __str__(self):
        return '(NUMBER, %s)' % str(self.value)

    def readable_str(self):
        return "%s" % str(self.value)


class Boolean:
    def __init__(self, value, lineno=None, is_public=None):
        self.value = value
        self.lineno = lineno
        self.is_public = is_public

    def is_public_exp(self):
        return self.is_public

    def __str__(self):
        return '(BOOLEAN, %s)' % str(self.value)

    def readable_str(self):
        if self.value:
            return "1"
        else:
            return "0"


class IfExpr:
    def __init__(self, cond, then_branch, else_branch, lineno=None, is_public=None):
        self.cond = cond
        self.then_branch = then_branch
        self.else_branch = else_branch
        self.lineno = lineno
        self.is_public = is_public

    def is_public_exp(self):
        return self.is_public

    def __str__(self):
        return '(IF-THEN-ELSE, %s, %s, %s)' % (str(self.cond), str(self.then_branch), str(self.else_branch))

    def readable_str(self):
        return '(if %s then %s else %s)' % (self.cond.readable_str(), self.then_branch.readable_str(), self.else_branch.readable_str())


class FuncCallExpr:
    def __init__(self, func_id, args, lineno=None, is_public=None):
        self.func_id = func_id
        self.args = args
        self.lineno = lineno
        self.is_public = is_public

    def is_public_exp(self):
        return self.is_public

    def __str__(self):
        return '(FUNCTION CALL, %s, [%s])' % (str(self.func_id), ', '.join([str(elem) for elem in self.args]))

    def readable_str(self):
        return '%s(%s)' % (self.func_id.readable_str(), ', '.join([elem.readable_str() for elem in self.args]))


class JumpIfFalseStm:
    def __init__(self, var, destination):
        self.var = var
        self.destination = destination

    def __str__(self):
        return "(JZ, %s, %s)" % (str(self.var), str(self.destination))

    def readable_str(self):
        return "JZ %s %s;" % (self.var.readable_str(), self.destination.readable_str())


class JumpUnconditionalStm:
    def __init__(self, destination):
        self.destination = destination

    def __str__(self):
        return "(JMP, %s)" % str(self.destination)

    def readable_str(self):
        return "JMP %s;" % self.destination.readable_str()


class IfResultId:
    def __init__(self, name):
        self.name = name

    def __str__(self):
        return "(IF-RESULT, %s)" % self.name

    def readable_str(self):
        return "%s (if-result)" % self.name


class ProgramPoint:
    def __init__(self, num):
        self.num = num

    def __str__(self):
        return "(PROGRAM POINT, %d)" % self.num

    def readable_str(self):
        return "ProgramPoint %d" % self.num


