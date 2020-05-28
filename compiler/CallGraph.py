import ASTnodes


class CallGraphNode:
    def __init__(self, func_name):
        self.func_name = func_name
        self.pt = set()
        self.visited = False
        self.visiting = False

    def resetDFS(self):     # If we should ever wish to perform multiple DFS runs
        self.visited = False
        self.visiting = False

    def __str__(self):
        return "Function '%s' pointsTo: [%s])" % (self.func_name, ', '.join([elem for elem in self.pt]))


class CallGraph:
    def __init__(self, prog):
        self.prog = prog
        self.cg = {}
        self.func_overload = False
        self.insert_cg_nodes()
        self.insert_cg_edges()

    def __str__(self):
        return '\n'.join([str(cg_node) for cg_node in self.cg.values()])

    def insert_cg_nodes(self):
        for func in self.prog.funcs:
            func_name = func.id.name
            if func_name in self.cg.keys():
                print("ERROR: Multiple functions with same name '%s'. Only one allowed." % func_name)
                self.func_overload = True
            if func_name == 'randombit' or func_name == 'randomnum':
                print("ERROR: Redefining built-in function '%s'." % func_name)
                self.func_overload = True
            self.cg[func_name] = CallGraphNode(func_name)
        self.cg['randomnum'] = CallGraphNode('randomnum')
        self.cg['randombit'] = CallGraphNode('randombit')

    def insert_cg_edges(self):
        for func in self.prog.funcs:
            for stm in func.body.stms:
                if not isinstance(stm, ASTnodes.AssignStm):
                    continue  # No expressions
                self.expr_add_cg_edges(stm.expr, func.id.name)
            if func.id.name != 'main':
                self.expr_add_cg_edges(func.body.expr, func.id.name)

    def expr_add_cg_edges(self, expr, func_name):
        if isinstance(expr, ASTnodes.Number) or isinstance(expr, ASTnodes.Boolean) or isinstance(expr, ASTnodes.Identifier):
            return

        if isinstance(expr, ASTnodes.Binop):
            self.expr_add_cg_edges(expr.left, func_name)
            self.expr_add_cg_edges(expr.right, func_name)
            return

        if isinstance(expr, ASTnodes.Uminus) or isinstance(expr, ASTnodes.Not) or isinstance(expr, ASTnodes.LeakExpr):
            self.expr_add_cg_edges(expr.sub_expr, func_name)
            return

        if isinstance(expr, ASTnodes.IfExpr):
            self.expr_add_cg_edges(expr.cond, func_name)
            self.expr_add_cg_edges(expr.then_branch, func_name)
            self.expr_add_cg_edges(expr.else_branch, func_name)
            return

        if not isinstance(expr, ASTnodes.FuncCallExpr):
            raise Exception("ERROR: Unexpected expression token found doing call graph construction")

        if expr.func_id.name == 'main':     # Only necessary if call comes from unreachable function...
            raise Exception("ERROR in line %d: Illegal function call to 'main'." % expr.lineno)

        if not expr.func_id.name in self.cg:
            raise Exception("ERROR in line %d: Calling undeclared function '%s'." % (expr.lineno, expr.func_id.name))

        self.cg[func_name].pt.add(expr.func_id.name)
        for arg in expr.args:
            self.expr_add_cg_edges(arg, func_name)
        return

    def has_recursion(self):
        result = self.has_recursion_dfs(self.cg['main'], [])
        for node in self.cg.values():
            node.resetDFS()
        return result

    def has_recursion_dfs(self, node, path):
        if node.visited:
            return False

        path.append(node.func_name)
        if node.visiting:
            print("ERROR: Recursion on path: %s" % ' --> '.join(path))
            return True
        node.visiting = True
        for next_node in node.pt:
            if self.has_recursion_dfs(self.cg[next_node], path):
                return True
        path.pop()
        node.visited = True
        return False
