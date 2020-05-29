from compiler import ASTnodes
from compiler.Lexer import tokens
from compiler.HelperFunctions import list_from_functional_list


# A program is a list of functions
def p_prog(p):
    """prog : funcs"""
    p[0] = ASTnodes.Program(funcs=list_from_functional_list(p[1]))



def p_funcs(p):
    """funcs : func funcs
             | empty"""
    if len(p) == 3:
        p[0] = (p[1], p[2])
    else:
        p[0] = None


def p_func(p):
    """func : ID arglist LBRACE funcbody RBRACE
            | MAIN LPAREN RPAREN LBRACE stms RBRACE"""
    if len(p) == 6:
        p[0] = ASTnodes.Function(id=ASTnodes.Identifier(p[1], p.lineno(1)),
                                 args=list_from_functional_list(p[2]),
                                 body=p[4],
                                 lineno=p.lineno(1))
    else:
        p[0] = ASTnodes.Function(id=ASTnodes.Identifier('main', p.lineno(1)),
                                 args=[],
                                 body=ASTnodes.FunctionBody(stms=list_from_functional_list(p[5], rev=True),
                                                            expr=None),
                                 lineno=p.lineno(1))


def p_funcbody(p):
    """funcbody : stms expression
                | expression"""
    if len(p) == 3:
        p[0] = ASTnodes.FunctionBody(stms=list_from_functional_list(p[1], rev=True), expr=p[2])
    else:
        p[0] = ASTnodes.FunctionBody(stms=[], expr=p[1])


def p_arglist(p):
    """arglist : LPAREN RPAREN
               | LPAREN arg args RPAREN"""
    if len(p) == 3:
        p[0] = None
    else:
        p[0] = (p[2], p[3])


def p_args(p):
    """args : COMMA arg args
            | empty"""
    if len(p) == 4:
        p[0] = (p[2], p[3])
    else:
        p[0] = None


def p_arg(p):
    """arg : ID"""
    p[0] = ASTnodes.Identifier(p[1], p.lineno(1))


def p_stms(p):
    """stms : stms stm
            | stm"""
    if len(p) == 3:
        p[0] = (p[2], p[1])
    else:
        p[0] = (p[1], None)


def p_stm(p):
    """stm : ID ASSIGN expression SEMICOLON
           | ID INPUT NUMBER COLON type SEMICOLON
           | ID OUTPUT ID SEMICOLON"""
    if p[2] == '=':
        p[0] = ASTnodes.AssignStm(var=ASTnodes.Identifier(p[1], p.lineno(1)),
                                  expr=p[3],
                                  lineno=p.lineno(1))
    elif p[2] == '<<':
        p[0] = ASTnodes.InputStm(var=ASTnodes.Identifier(p[1], p.lineno(1)),
                                 input_provider=ASTnodes.Number(p[3], p.lineno(3)),
                                 type=p[5],
                                 lineno=p.lineno(1))
    else:
        p[0] = ASTnodes.OutputStm(output_var=ASTnodes.Identifier(p[1], p.lineno(1)),
                                  result_name=ASTnodes.Identifier(p[3], p.lineno(3)),
                                  lineno=p.lineno(1))


def p_type(p):
    """type : BOOL
            | NUM"""
    p[0] = p[1]


precedence = (
    ('right', 'LEAK'),
    ('left', 'OR'),
    ('left', 'AND'),
    ('right', 'NOT'),
    ('nonassoc', 'EQUALS', 'NEQ', 'LT', 'GT', 'LTE', 'GTE'),
    ('left', 'PLUS', 'MINUS'),
    ('left', 'TIMES', 'DIVIDE'),
    ('right', 'UMINUS')
)


def p_expression_arithmetic(p):
    """expression : expression PLUS expression
                  | expression MINUS expression
                  | expression TIMES expression
                  | expression DIVIDE expression
                  | expression EQUALS expression
                  | expression NEQ expression
                  | expression LT expression
                  | expression GT expression
                  | expression LTE expression
                  | expression GTE expression
                  | expression OR expression
                  | expression AND expression
                  | TRUE
                  | FALSE
                  | NUMBER
                  | MINUS expression %prec UMINUS
                  | NOT expression"""
    if len(p) == 4:
        p[0] = ASTnodes.Binop(op=p[2],
                              left=p[1],
                              right=p[3],
                              lineno=p.lineno(2))
    elif len(p) == 3:
        if p[1] == '-':
            p[0] = ASTnodes.Uminus(p[2], p.lineno(1))
        else:
            p[0] = ASTnodes.Not(p[2], p.lineno(1))
    else:
        if p[1] == 'true':
            p[0] = ASTnodes.Boolean(True, p.lineno(1))
        elif p[1] == 'false':
            p[0] = ASTnodes.Boolean(False, p.lineno(1))
        else:
            p[0] = ASTnodes.Number(p[1], p.lineno(1))


def p_expression_leak(p):
    """expression : LEAK expression"""
    p[0] = ASTnodes.LeakExpr(expr=p[2],
                             lineno=p.lineno(1))


def p_expression_id(p):
    """expression : ID"""
    p[0] = ASTnodes.Identifier(p[1], p.lineno(1))


def p_expression_paren(p):
    """expression : LPAREN expression RPAREN"""
    p[0] = p[2]


def p_expression_if(p):
    """expression : IF LPAREN expression RPAREN LBRACE expression RBRACE ELSE LBRACE expression RBRACE"""
    # if (a == b) { 3 } else { 5 + 4 }
    p[0] = ASTnodes.IfExpr(cond=p[3],
                           then_branch=p[6],
                           else_branch=p[10],
                           lineno=p.lineno(1))


def p_expression_func_call(p):
    """expression : ID LPAREN exps RPAREN
                  | ID LPAREN RPAREN"""
    if len(p) == 4:
        p[0] = ASTnodes.FuncCallExpr(func_id=ASTnodes.Identifier(p[1], p.lineno(1)),
                                     args=[],
                                     lineno=p.lineno(1))
    else:
        p[0] = ASTnodes.FuncCallExpr(func_id=ASTnodes.Identifier(p[1], p.lineno(1)),
                                     args=list_from_functional_list(p[3]),
                                     lineno=p.lineno(1))


def p_exps(p):
    """exps : expression
            | expression COMMA exps"""
    if len(p) == 2:
        p[0] = (p[1], None)
    else:
        p[0] = (p[1], p[3])


# Rule for handling empty productions
def p_empty(p):
    """empty :"""
    pass


def p_error(p):
    print("Syntax error in (or just before) line %d at symbol: '%s'" % (p.lineno, p.value))
    p.lexer.error = True
