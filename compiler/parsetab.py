
# parsetab.py
# This file is automatically generated. Do not edit.
# pylint: disable=W,C,R
_tabversion = '3.10'

_lr_method = 'LALR'

_lr_signature = 'rightLEAKleftORleftANDrightNOTnonassocEQUALSNEQLTGTLTEGTEleftPLUSMINUSleftTIMESDIVIDErightUMINUSAND ASSIGN BOOL COLON COMMA DIVIDE ELSE EQUALS FALSE GT GTE ID IF INPUT LBRACE LEAK LPAREN LSQBRACKET LT LTE MAIN MINUS NEQ NOT NUM NUMBER OR OUTPUT PLUS RBRACE RPAREN RSQBRACKET SEMICOLON TIMES TRUEprog : funcsfuncs : func funcs\n             | emptyfunc : ID arglist LBRACE funcbody RBRACE\n            | MAIN LPAREN RPAREN LBRACE stms RBRACEfuncbody : stms expression\n                | expressionarglist : LPAREN RPAREN\n               | LPAREN arg args RPARENargs : COMMA arg args\n            | emptyarg : IDstms : stms stm\n            | stmstm : ID ASSIGN expression SEMICOLON\n           | ID INPUT NUMBER COLON type SEMICOLON\n           | ID OUTPUT ID SEMICOLONtype : BOOL\n            | NUMexpression : expression PLUS expression\n                  | expression MINUS expression\n                  | expression TIMES expression\n                  | expression DIVIDE expression\n                  | expression EQUALS expression\n                  | expression NEQ expression\n                  | expression LT expression\n                  | expression GT expression\n                  | expression LTE expression\n                  | expression GTE expression\n                  | expression OR expression\n                  | expression AND expression\n                  | TRUE\n                  | FALSE\n                  | NUMBER\n                  | MINUS expression %prec UMINUS\n                  | NOT expressionexpression : LEAK expressionexpression : IDexpression : LPAREN expression RPARENexpression : IF LPAREN expression RPAREN LBRACE expression RBRACE ELSE LBRACE expression RBRACEexpression : ID LPAREN exps RPAREN\n                  | ID LPAREN RPARENexps : expression\n            | expression COMMA expsempty :'
    
_lr_action_items = {'ID':([0,3,9,11,18,20,21,25,26,27,30,32,33,34,36,37,39,40,41,42,43,44,45,46,47,48,49,50,51,57,60,83,85,86,88,94,95,99,],[5,5,14,16,16,-14,53,53,53,53,14,61,53,53,67,-4,-13,53,53,53,53,53,53,53,53,53,53,53,53,53,61,-5,53,-15,-17,53,-16,53,]),'MAIN':([0,3,37,83,],[6,6,-4,-5,]),'$end':([0,1,2,3,4,7,37,83,],[-45,0,-1,-45,-3,-2,-4,-5,]),'LPAREN':([5,6,11,16,18,20,21,25,26,27,28,33,34,39,40,41,42,43,44,45,46,47,48,49,50,51,53,57,85,86,88,94,95,99,],[9,10,27,33,27,-14,27,27,27,27,57,27,27,-13,27,27,27,27,27,27,27,27,27,27,27,27,33,27,27,-15,-17,27,-16,27,]),'LBRACE':([8,12,15,58,89,98,],[11,-8,32,-9,94,99,]),'RPAREN':([9,10,13,14,22,23,24,29,31,33,52,53,54,55,56,59,62,63,64,68,69,70,71,72,73,74,75,76,77,78,79,80,81,82,84,90,101,],[12,15,-45,-12,-32,-33,-34,58,-11,63,-35,-38,-36,-37,80,-45,84,-42,-43,-20,-21,-22,-23,-24,-25,-26,-27,-28,-29,-30,-31,-39,89,-10,-41,-44,-40,]),'TRUE':([11,18,20,21,25,26,27,33,34,39,40,41,42,43,44,45,46,47,48,49,50,51,57,85,86,88,94,95,99,],[22,22,-14,22,22,22,22,22,22,-13,22,22,22,22,22,22,22,22,22,22,22,22,22,22,-15,-17,22,-16,22,]),'FALSE':([11,18,20,21,25,26,27,33,34,39,40,41,42,43,44,45,46,47,48,49,50,51,57,85,86,88,94,95,99,],[23,23,-14,23,23,23,23,23,23,-13,23,23,23,23,23,23,23,23,23,23,23,23,23,23,-15,-17,23,-16,23,]),'NUMBER':([11,18,20,21,25,26,27,33,34,35,39,40,41,42,43,44,45,46,47,48,49,50,51,57,85,86,88,94,95,99,],[24,24,-14,24,24,24,24,24,24,66,-13,24,24,24,24,24,24,24,24,24,24,24,24,24,24,-15,-17,24,-16,24,]),'MINUS':([11,16,18,19,20,21,22,23,24,25,26,27,33,34,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,63,64,65,68,69,70,71,72,73,74,75,76,77,78,79,80,81,84,85,86,88,94,95,96,99,100,101,],[21,-38,21,41,-14,21,-32,-33,-34,21,21,21,21,21,41,-13,21,21,21,21,21,21,21,21,21,21,21,21,-35,-38,41,41,41,21,-42,41,41,-20,-21,-22,-23,41,41,41,41,41,41,41,41,-39,41,-41,21,-15,-17,21,-16,41,21,41,-40,]),'NOT':([11,18,20,21,25,26,27,33,34,39,40,41,42,43,44,45,46,47,48,49,50,51,57,85,86,88,94,95,99,],[25,25,-14,25,25,25,25,25,25,-13,25,25,25,25,25,25,25,25,25,25,25,25,25,25,-15,-17,25,-16,25,]),'LEAK':([11,18,20,21,25,26,27,33,34,39,40,41,42,43,44,45,46,47,48,49,50,51,57,85,86,88,94,95,99,],[26,26,-14,26,26,26,26,26,26,-13,26,26,26,26,26,26,26,26,26,26,26,26,26,26,-15,-17,26,-16,26,]),'IF':([11,18,20,21,25,26,27,33,34,39,40,41,42,43,44,45,46,47,48,49,50,51,57,85,86,88,94,95,99,],[28,28,-14,28,28,28,28,28,28,-13,28,28,28,28,28,28,28,28,28,28,28,28,28,28,-15,-17,28,-16,28,]),'COMMA':([13,14,22,23,24,52,53,54,55,59,63,64,68,69,70,71,72,73,74,75,76,77,78,79,80,84,101,],[30,-12,-32,-33,-34,-35,-38,-36,-37,30,-42,85,-20,-21,-22,-23,-24,-25,-26,-27,-28,-29,-30,-31,-39,-41,-40,]),'PLUS':([16,19,22,23,24,38,52,53,54,55,56,63,64,65,68,69,70,71,72,73,74,75,76,77,78,79,80,81,84,96,100,101,],[-38,40,-32,-33,-34,40,-35,-38,40,40,40,-42,40,40,-20,-21,-22,-23,40,40,40,40,40,40,40,40,-39,40,-41,40,40,-40,]),'TIMES':([16,19,22,23,24,38,52,53,54,55,56,63,64,65,68,69,70,71,72,73,74,75,76,77,78,79,80,81,84,96,100,101,],[-38,42,-32,-33,-34,42,-35,-38,42,42,42,-42,42,42,42,42,-22,-23,42,42,42,42,42,42,42,42,-39,42,-41,42,42,-40,]),'DIVIDE':([16,19,22,23,24,38,52,53,54,55,56,63,64,65,68,69,70,71,72,73,74,75,76,77,78,79,80,81,84,96,100,101,],[-38,43,-32,-33,-34,43,-35,-38,43,43,43,-42,43,43,43,43,-22,-23,43,43,43,43,43,43,43,43,-39,43,-41,43,43,-40,]),'EQUALS':([16,19,22,23,24,38,52,53,54,55,56,63,64,65,68,69,70,71,72,73,74,75,76,77,78,79,80,81,84,96,100,101,],[-38,44,-32,-33,-34,44,-35,-38,44,44,44,-42,44,44,-20,-21,-22,-23,None,None,None,None,None,None,44,44,-39,44,-41,44,44,-40,]),'NEQ':([16,19,22,23,24,38,52,53,54,55,56,63,64,65,68,69,70,71,72,73,74,75,76,77,78,79,80,81,84,96,100,101,],[-38,45,-32,-33,-34,45,-35,-38,45,45,45,-42,45,45,-20,-21,-22,-23,None,None,None,None,None,None,45,45,-39,45,-41,45,45,-40,]),'LT':([16,19,22,23,24,38,52,53,54,55,56,63,64,65,68,69,70,71,72,73,74,75,76,77,78,79,80,81,84,96,100,101,],[-38,46,-32,-33,-34,46,-35,-38,46,46,46,-42,46,46,-20,-21,-22,-23,None,None,None,None,None,None,46,46,-39,46,-41,46,46,-40,]),'GT':([16,19,22,23,24,38,52,53,54,55,56,63,64,65,68,69,70,71,72,73,74,75,76,77,78,79,80,81,84,96,100,101,],[-38,47,-32,-33,-34,47,-35,-38,47,47,47,-42,47,47,-20,-21,-22,-23,None,None,None,None,None,None,47,47,-39,47,-41,47,47,-40,]),'LTE':([16,19,22,23,24,38,52,53,54,55,56,63,64,65,68,69,70,71,72,73,74,75,76,77,78,79,80,81,84,96,100,101,],[-38,48,-32,-33,-34,48,-35,-38,48,48,48,-42,48,48,-20,-21,-22,-23,None,None,None,None,None,None,48,48,-39,48,-41,48,48,-40,]),'GTE':([16,19,22,23,24,38,52,53,54,55,56,63,64,65,68,69,70,71,72,73,74,75,76,77,78,79,80,81,84,96,100,101,],[-38,49,-32,-33,-34,49,-35,-38,49,49,49,-42,49,49,-20,-21,-22,-23,None,None,None,None,None,None,49,49,-39,49,-41,49,49,-40,]),'OR':([16,19,22,23,24,38,52,53,54,55,56,63,64,65,68,69,70,71,72,73,74,75,76,77,78,79,80,81,84,96,100,101,],[-38,50,-32,-33,-34,50,-35,-38,-36,50,50,-42,50,50,-20,-21,-22,-23,-24,-25,-26,-27,-28,-29,-30,-31,-39,50,-41,50,50,-40,]),'AND':([16,19,22,23,24,38,52,53,54,55,56,63,64,65,68,69,70,71,72,73,74,75,76,77,78,79,80,81,84,96,100,101,],[-38,51,-32,-33,-34,51,-35,-38,-36,51,51,-42,51,51,-20,-21,-22,-23,-24,-25,-26,-27,-28,-29,51,-31,-39,51,-41,51,51,-40,]),'RBRACE':([16,17,19,20,22,23,24,38,39,52,53,54,55,60,63,68,69,70,71,72,73,74,75,76,77,78,79,80,84,86,88,95,96,100,101,],[-38,37,-7,-14,-32,-33,-34,-6,-13,-35,-38,-36,-37,83,-42,-20,-21,-22,-23,-24,-25,-26,-27,-28,-29,-30,-31,-39,-41,-15,-17,-16,97,101,-40,]),'ASSIGN':([16,61,],[34,34,]),'INPUT':([16,61,],[35,35,]),'OUTPUT':([16,61,],[36,36,]),'SEMICOLON':([22,23,24,52,53,54,55,63,65,67,68,69,70,71,72,73,74,75,76,77,78,79,80,84,91,92,93,101,],[-32,-33,-34,-35,-38,-36,-37,-42,86,88,-20,-21,-22,-23,-24,-25,-26,-27,-28,-29,-30,-31,-39,-41,95,-18,-19,-40,]),'COLON':([66,],[87,]),'BOOL':([87,],[92,]),'NUM':([87,],[93,]),'ELSE':([97,],[98,]),}

_lr_action = {}
for _k, _v in _lr_action_items.items():
   for _x,_y in zip(_v[0],_v[1]):
      if not _x in _lr_action:  _lr_action[_x] = {}
      _lr_action[_x][_k] = _y
del _lr_action_items

_lr_goto_items = {'prog':([0,],[1,]),'funcs':([0,3,],[2,7,]),'func':([0,3,],[3,3,]),'empty':([0,3,13,59,],[4,4,31,31,]),'arglist':([5,],[8,]),'arg':([9,30,],[13,59,]),'funcbody':([11,],[17,]),'stms':([11,32,],[18,60,]),'expression':([11,18,21,25,26,27,33,34,40,41,42,43,44,45,46,47,48,49,50,51,57,85,94,99,],[19,38,52,54,55,56,64,65,68,69,70,71,72,73,74,75,76,77,78,79,81,64,96,100,]),'stm':([11,18,32,60,],[20,39,20,39,]),'args':([13,59,],[29,82,]),'exps':([33,85,],[62,90,]),'type':([87,],[91,]),}

_lr_goto = {}
for _k, _v in _lr_goto_items.items():
   for _x, _y in zip(_v[0], _v[1]):
       if not _x in _lr_goto: _lr_goto[_x] = {}
       _lr_goto[_x][_k] = _y
del _lr_goto_items
_lr_productions = [
  ("S' -> prog","S'",1,None,None,None),
  ('prog -> funcs','prog',1,'p_prog','Parser.py',8),
  ('funcs -> func funcs','funcs',2,'p_funcs','Parser.py',14),
  ('funcs -> empty','funcs',1,'p_funcs','Parser.py',15),
  ('func -> ID arglist LBRACE funcbody RBRACE','func',5,'p_func','Parser.py',23),
  ('func -> MAIN LPAREN RPAREN LBRACE stms RBRACE','func',6,'p_func','Parser.py',24),
  ('funcbody -> stms expression','funcbody',2,'p_funcbody','Parser.py',39),
  ('funcbody -> expression','funcbody',1,'p_funcbody','Parser.py',40),
  ('arglist -> LPAREN RPAREN','arglist',2,'p_arglist','Parser.py',48),
  ('arglist -> LPAREN arg args RPAREN','arglist',4,'p_arglist','Parser.py',49),
  ('args -> COMMA arg args','args',3,'p_args','Parser.py',57),
  ('args -> empty','args',1,'p_args','Parser.py',58),
  ('arg -> ID','arg',1,'p_arg','Parser.py',66),
  ('stms -> stms stm','stms',2,'p_stms','Parser.py',71),
  ('stms -> stm','stms',1,'p_stms','Parser.py',72),
  ('stm -> ID ASSIGN expression SEMICOLON','stm',4,'p_stm','Parser.py',80),
  ('stm -> ID INPUT NUMBER COLON type SEMICOLON','stm',6,'p_stm','Parser.py',81),
  ('stm -> ID OUTPUT ID SEMICOLON','stm',4,'p_stm','Parser.py',82),
  ('type -> BOOL','type',1,'p_type','Parser.py',99),
  ('type -> NUM','type',1,'p_type','Parser.py',100),
  ('expression -> expression PLUS expression','expression',3,'p_expression_arithmetic','Parser.py',117),
  ('expression -> expression MINUS expression','expression',3,'p_expression_arithmetic','Parser.py',118),
  ('expression -> expression TIMES expression','expression',3,'p_expression_arithmetic','Parser.py',119),
  ('expression -> expression DIVIDE expression','expression',3,'p_expression_arithmetic','Parser.py',120),
  ('expression -> expression EQUALS expression','expression',3,'p_expression_arithmetic','Parser.py',121),
  ('expression -> expression NEQ expression','expression',3,'p_expression_arithmetic','Parser.py',122),
  ('expression -> expression LT expression','expression',3,'p_expression_arithmetic','Parser.py',123),
  ('expression -> expression GT expression','expression',3,'p_expression_arithmetic','Parser.py',124),
  ('expression -> expression LTE expression','expression',3,'p_expression_arithmetic','Parser.py',125),
  ('expression -> expression GTE expression','expression',3,'p_expression_arithmetic','Parser.py',126),
  ('expression -> expression OR expression','expression',3,'p_expression_arithmetic','Parser.py',127),
  ('expression -> expression AND expression','expression',3,'p_expression_arithmetic','Parser.py',128),
  ('expression -> TRUE','expression',1,'p_expression_arithmetic','Parser.py',129),
  ('expression -> FALSE','expression',1,'p_expression_arithmetic','Parser.py',130),
  ('expression -> NUMBER','expression',1,'p_expression_arithmetic','Parser.py',131),
  ('expression -> MINUS expression','expression',2,'p_expression_arithmetic','Parser.py',132),
  ('expression -> NOT expression','expression',2,'p_expression_arithmetic','Parser.py',133),
  ('expression -> LEAK expression','expression',2,'p_expression_leak','Parser.py',154),
  ('expression -> ID','expression',1,'p_expression_id','Parser.py',160),
  ('expression -> LPAREN expression RPAREN','expression',3,'p_expression_paren','Parser.py',165),
  ('expression -> IF LPAREN expression RPAREN LBRACE expression RBRACE ELSE LBRACE expression RBRACE','expression',11,'p_expression_if','Parser.py',170),
  ('expression -> ID LPAREN exps RPAREN','expression',4,'p_expression_func_call','Parser.py',179),
  ('expression -> ID LPAREN RPAREN','expression',3,'p_expression_func_call','Parser.py',180),
  ('exps -> expression','exps',1,'p_exps','Parser.py',192),
  ('exps -> expression COMMA exps','exps',3,'p_exps','Parser.py',193),
  ('empty -> <empty>','empty',0,'p_empty','Parser.py',202),
]