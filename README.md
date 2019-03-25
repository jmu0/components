# components library for js and go
- ui components
- data api(go)
- routing on server(go) and client(js)
- builds same html on client(js) and server(go)


## app.json
- Structure app / nesting components / pages
- Routing for pages
- ComponentPath: load components in this path
- Scripts: array of scripts to load. append to body when in debug mode
- concatenate, minify and gzip scripts on server start, serve single js file.

## components
- load component from path
- name=folder name
### get.sql
- stores sql statement to GET data
- adds route for data when get.sql is found
### templates
- *.html are loaded as templates
- every component has a TemplateManager
- adds route for /component/[name]
- handle POST and DELETE requests using dbModel (use data.json file to store db/table?)
### less
- build tool adds all .less files to /static/css/components.less (run: build less)
### js
- adds route for all .js files
- if filename != component name => serves [componentname].[filename]
- when in debug mode, puts all .js files in script tags and appends to body when serving a page
- TODO: concatenate, minify and gzip all scripts files on server start, serve single js file.
- build tool adds symlinks for all script files to /static/js/ for debugging in vscode (run: build js debug)
- build tool builds single minified /static/js/[appname].js file for all components. (run: build js)
## authentication
- TODO: use jwt for authentication. set auth field in app.json

