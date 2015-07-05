package main

type HTTPModifierConfig struct {
    urlRegexp            HTTPUrlRegexp
    urlRewrite           UrlRewriteMap
    headerFilters        HTTPHeaderFilters
    headerHashFilters    HTTPHeaderHashFilters

    headers HTTPHeaders
    methods HTTPMethods
}