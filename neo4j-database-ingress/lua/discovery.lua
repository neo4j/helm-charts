local http = require('http')
local json = require('json')

local function neo4j_discovery(applet)
    local req = http.request.parse(applet)

    local req_host = req:get_header("host")
    if not req_host then
        core.Alert("request doesn't have host header!?")
        return
    end
    for k, v in req:get_headers() do
        core.Debug(k .. ": " .. v)
    end
    local req_host_port = ""
    local req_port = req:get_header("x-forwarded-port")
    -- because the js driver will provide a default port if we don't!
    if req_host:find(":", 1, true) ~= nil then
        req_host_and_port = req_host
        local host_match, _ = string.gmatch(req_host, '([^:]+)', 2)
        req_host = host_match()
        req_port = host_match()
    elseif req:get_header("x-scheme") == "https" and req:get_header("x-forwarded-port") == "443" then
        req_host_and_port = req_host .. ":443"
    elseif req:get_header("x-scheme") == "http" and req:get_header("x-forwarded-port") == "80" then
        req_host_and_port = req_host .. ":80"
    end
    core.Info(string.format("translating discovery request with req_host_and_port : %s", req_host_and_port))

    -- look for a particular backend named "neo4j-http"
    local http_backend = core.backends["neo4j-http"]
    if not http_backend then
        core.Alert("cannot find backend named 'neo4j-http'")
        return
    end

    -- get the first server in our backend
    local server = nil
    for k, v in pairs(http_backend.servers) do
        server = v
        break
    end
    local host = server:get_addr()
    if not host then
        core.Alert(string.format("can't get a host value for server %s", server))
        return
    end
    local res, err = http.get { url = string.format("http://%s", host),
                                headers = { host = host,
                                            ["accept"] = { "application/json" },
                                            ["connection"] = { "close" } }
    }
    if res then
        for k, v in res:get_headers() do
            core.Debug(k .. ": " .. v)
        end
    else
        core.Debug(err)
    end
    local response_body = json.decode(res.content)
    local proxy_response = {}
    local ip_address_match, _ = string.gmatch(host, '([^:]+)', 2)
    local ip = ip_address_match()
    local port = ip_address_match()

    for index, value in pairs(response_body) do
        if not (index == "auth_config") then
            proxy_response[index] = string.gsub(value, ip, req_host)
            proxy_response[index] = string.gsub(proxy_response[index], "7474", req_port)
            proxy_response[index] = string.gsub(proxy_response[index], "7687", req_port)
            if req:get_header("x-forwarded-port") == "443" then
                proxy_response[index] = string.gsub(proxy_response[index], "http", "https")
                proxy_response[index] = string.gsub(proxy_response[index], "bolt", "bolt+s")
                proxy_response[index] = string.gsub(proxy_response[index], "neo4j", "neo4j+s")
            end
            core.Debug(value)
            core.Debug(proxy_response[index])
        end
    end
    http.response.create {
        status_code = 200,
        content = json.encode(proxy_response)}:send(applet)

end
core.register_service('neo4j_discovery', 'http', neo4j_discovery)