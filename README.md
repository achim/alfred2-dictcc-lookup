alfred2-dictcc-lookup
=====================

Alfred 2 workflow for querying [dict.cc](http://www.dict.cc), with support for suggestions and previews.

**NOTE:** This is my first exposure to Go and should be considered a toy project. If you're looking for something immediately usable, there are a couple of projects doing essentially the same thing,
like https://github.com/elkatwork/alfred-dict.cc-workflow

Flag icons are CC BY-SA 3.0 Matthias Slovig, http://flags.blogpotato.de

Installation (using [Homebrew](http://brew.sh)):
------------------------------------------------

1. Install Go:

    * `brew install go`
	* Set up your enviroment as described in http://golang.org/doc/install#install
    * 
    
2. Install Kyoto Cabinet:

	`brew install kyoto-cabinet`

3. Download and install the executable:

	`go get github.com/achim/alfred2-dictcc-lookup`

4. Download and import the workflow:

   * `curl -LO 'https://github.com/achim/alfred2-dictcc-lookup/raw/master/alfred2-dictcc-lookup.alfredworkflow'`
   * `open open alfred2-dictcc-lookup.alfredworkflow`

