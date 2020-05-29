# module: Lexer.py

reserved = {
    'if': 'IF',
    'else': 'ELSE',
    'leak': 'LEAK',
    'main': 'MAIN',
    'true': 'TRUE',
    'false': 'FALSE',
    'bool': 'BOOL',
    'num': 'NUM'
}

# List of token names.   This is always required
tokens = [
             'NUMBER',
             'PLUS',
             'MINUS',
             'TIMES',
             'DIVIDE',
             'AND',
             'OR',
             'NOT',
             'ASSIGN',
             'INPUT',
             'OUTPUT',
             'EQUALS',
             'NEQ',
             'LT',
             'GT',
             'LTE',
             'GTE',
             'LPAREN',
             'RPAREN',
             'LBRACE',
             'RBRACE',
             'LSQBRACKET',
             'RSQBRACKET',
             'COLON',
             'SEMICOLON',
             'COMMA',
             'ID'
         ] + list(reserved.values())

# Regular expression rules for simple tokens
t_PLUS = r'\+'
t_MINUS = r'-'
t_TIMES = r'\*'
t_DIVIDE = r'/'
t_AND = r'&&'
t_OR = r'\|\|'
t_NOT = r'!'
t_ASSIGN = r'='
t_INPUT = r'<<'
t_OUTPUT = r'>>'
t_EQUALS = r'=='
t_NEQ = r'!='
t_LT = r'<'
t_GT = r'>'
t_LTE = r'<='
t_GTE = r'>='
t_LPAREN = r'\('
t_RPAREN = r'\)'
t_LBRACE = r'\{'
t_RBRACE = r'\}'
t_LSQBRACKET = r'\['
t_RSQBRACKET = r'\]'
t_COLON = r':'
t_SEMICOLON = r';'
t_COMMA = r','


# A regular expression rule with some action code
def t_NUMBER(t):
    r'\d+'
    t.value = int(t.value)
    return t


def t_ID(t):
    r'[a-zA-Z][a-zA-Z_0-9]*'
    t.type = reserved.get(t.value, 'ID')  # Check for reserved words
    return t


# Define a rule so we can track line numbers
def t_newline(t):
    r'\n+'
    t.lexer.lineno += len(t.value)


# A string containing ignored characters (spaces and tabs)
t_ignore = ' \t'


# Error handling rule
def t_error(t):
    print("LEXING ERROR: Illegal character in line %d: '%s'" % (t.lexer.lineno, t.value[0]))
    t.lexer.error = True
    t.lexer.skip(1)
