# HTExtractor
Simple http data extractor.
 
# Usage
## Test run
1. Create json file like `test/test.json`
2. Compile or run `test/main.go` with path to config file as argument

## Configuration
Extractor needs instructions or _actions_ to extract data from web page, base address and search context of web page according to base address.

Base address (`baseUrl`) - combination of `schema://[[login][:password]@]host[:port]`, schema can be `http` or `https`.

Search (`search`) - some text to search, can be empty, used in placeholder `${search}`.

_Actions_ are list of actions to perform get, search and extract some data.

## Actions
Single action is a struct with 2 parameters:

 - action - one of predefined function
 - param - specific parameter for function

Predefined actions:

 - `go` - just get data from URL (HTTP GET), `param` - is context template respectively to `baseUrl` to get data from.
 If there is no HTTP or carrier errors, next action is called
 Possible placeholders for `param`:
	- `${arg}` - value received from parent call (if any)
	- `${search}` - text value of `search` argument
	- `${selector}` - key extracted value, received from parent call (if any)
 - `extract` - extract substring from data, recieved from parent call with regexp provided in `param`. 
 If there is more than one match, action iterates over every match and send it to next action until the end, 
 or until first `findFirst` subaction is reached.
 If there is more than one group in match, action iterates over **every** group, providing group name or (if name not set) 
 group number to next action until the end.
 Possible placeholders for `param`:
	- `${search}` - text value of `search` argument
	- `${selector}` - key extracted value, received from parent call (if any)
 - `findAll` - checks data, recieved from parent call, if it contains data with regexp, provided in `param`, or if `param` is empty, 
 just checks that data not empty. If check succeded, next action is called. Possible placeholders for `param`:
	- `${search}` - text value of `search` argument
	- `${selector}` - key extracted value, received from parent call (if any)
 - `findFirst` - same as `findAll`, but it stops iteration of `extract` action after if it finds matched data
 - `store` - stores data received from parent call in the map with `selector` key or, if `param` is not empty, `param` + `selector`ю
 

Actions are executing sequentially,
but every next action is executing inside current action. If we have `go - extract - findFirst - store` sequence, it means, that
data which have been recieved by `go` action transmitted to `extract` action, then every data, that extracted by `extract` transmitted to
`findFirst`, and if `findFirst` susseded, `store` is called. 

## Example 
If we have next actions with params:

```json
{
	"search": "some value",
	"actions": [
		{
			"action": "go",
			"param": "/catalog"
		},
		{
			"action": "extract",
			"param": "<p class=\"catalog_info_name\">.*?<a .*?href=\"(.*?)\".*?>"
		},
		{
			"action": "go",
			"param": "/releases/${arg}"
		},
		{
			"action": "findFirst",
			"param": "<a class=\"button button_black\" href=\"\\Q${search}\\E\">"
		},
		{
			"action": "extract",
			"param": "<div class=\"title\">.*?<span>(?P<name>.*?)<\\/span>|<div class=\"description\">.*?<span>(?P<desc>.*?)<\\/span>"
		},
		{
			"action": "store",
			"param": "data."
		}
	]
}
```

1. Program goes to `/catalog` context of `baseUrl` site, gets all page content (let's say simple html page),
and sends it to `extract` action as argument
2. Program extracts all substrings of data from `go` action, that match `<p class="catalog_info_name">.*?<a .*?href="(?P<url>.*?)".*?>` regexp and for _every_ match, calls
`go` action with substring as an argument (let's say `release1`, then `release2`) and `1` as key (selector)
3. Program goes to page of `baseUrl` site, with context `/releases/release1` (`${arg}` placeholder in `param`),
gets all page data and sends it to `findFirst` action as argument, then again for `/releases/release2`, if `findFirst`'s subaction did not executed
4. Program checks if data from `go` action contains substring with `<a class="button button_black" href="\Qsome value\E">` (`${search}` placeholder in `param`) regexp,
if so, calls `extract` with data, recieved from `go` action
5. Program extracts substring from data that match regexp and calls `store` with `name` as key (selector), and then next substring with `desc` key (if any).
6. Program stores data from `extract` action in map with key `data.name` or `data.desc`.

