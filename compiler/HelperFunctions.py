def list_from_functional_list(functional_list, rev=False):
    my_list = []
    while functional_list is not None:
        my_list.append(functional_list[0])
        functional_list = functional_list[1]
    if rev:
        my_list.reverse()
    return my_list


def type_string(expr):
    if expr.is_public is None:
        return ""
    elif expr.is_public:
        return ":P"
    else:
        return ":S"
