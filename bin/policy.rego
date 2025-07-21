package ratify.policy
default valid := false

valid {
    not failed_verify(input)
}

failed_verify(reports) {
    [path, value] := walk(reports)
    value == false
    path[count(path) - 1] == "isSuccess"
}

failed_verify(reports) {
    [path, value] := walk(reports)
    path[count(path) - 1] == "verifierReports"
    count(value) == 0
}
