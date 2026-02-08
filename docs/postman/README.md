# Postman 测试 TR-069 (CWMP) 7547

## 文件

- Collection：`tr069-cwmp-7547.postman_collection.json`
- Environment：`tr069-cwmp-7547.postman_environment.json`

## 导入

1. 打开 Postman
2. Import → 选择上述两个 JSON 文件导入
3. 右上角环境下拉框选择 `TR069 CWMP Local`

## 运行用例（最小闭环）

按顺序执行：

1. `01 - Inform (BOOT)`
2. `02 - Poll (empty body)`
3. `03 - GPV Response (SerialNumber)`
4. `04 - Poll again (should be 204 if no pending cmd)`

其中第 2 步会自动从响应里提取 `<ID>...</ID>` 并写入环境变量 `cwmpId`，第 3 步会自动使用它。

## 关键点

- URL：默认 `http://127.0.0.1:7547/`，在环境变量 `cwmpBaseUrl` 可修改
- Header：所有请求都使用 `Content-Type: text/xml; charset=utf-8`
- Cookie：Postman 默认会自动维护 Cookie（用于 `tr069_session` 会话粘连）。如果你关闭了 Cookie 管理，需要打开它，否则第 2/3/4 步可能无法延续会话

## 预期结果

- Inform 返回 200，响应包含 `InformResponse`
- Poll 返回 200，响应包含 `GetParameterValues`，并包含新的 `<ID>...</ID>`
- GPV Response 通常返回 204（没有待下发命令时），或返回 200 并下发下一条 TR-069 Request（有命令待下发时）

