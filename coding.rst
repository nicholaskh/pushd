===============
Coding Standard
===============

.. contents:: Table Of Contents
.. section-numbering::

Overview
========

Don't make me think!


Dev Env
=======

- go
  go-1.4

- mongodb

Encoding
========

go file
########

utf-8

db table
########

utf-8

request/response
################

utf-8

Golang
===

design
######

- scope and visibility

  采用“作用域小”优先原则

  prefer private to public


- every struct should has a constructor

  ::

    struct testStruct {}
    func NewTestStruct () *testStruct

- prefer array mapping to switch clause

  too many cases is hard to read, but in array, they
  are straightforward.

- var declaration should be closest to its first reference

- class only talk to immediate friends
  
  never talk to strangers

- never copy extra variables

  especially when a var is referenced only once

  ::

    // bad
    desc := strip_tags(description);
    fmt.Println(desc);

    // good
    fmt.Println(strip_tags(description));

format
######

- always add a space between the keyword and a operator

  ::

    a = a + 1; // good
    a = a+1;   // bad

- empty line
  - an empty line is reserved for seperation of different logical unit
    
    never overuse empty line

  - between method/function blocks
    
    there will be 1 and only 1 empty line

- indent

  use tab or just use gofmt

- avoid line over 120 chars

- beginning bracelet will never occupy a whole line

  ::

    func foo() {    // do like this

    func foo() 
    {               // never do like this

- never use the following tags in file header
  @author

naming
######

- camel case names

  used for class name, var name, method name

- lower case connected with underscore names

  used for file name

- never use var name that ends with digits or new/old

  ::

    user2 := ''; // bad

    ipNew := ''; // bad


- use adjective for interfaces

  ::

    type Cacheable interface {}

- conventions

  - ModelClassName = {TableName} + 'Model'
    e,g. UserInfoModel

  - DataTableClassName = {TableName} + 'Table'
    e,g. UserInfoTable

- const use upper case with underscore connection

- do not reinvent an abbreviation unless it is really well known

comment
#######

It's a supplement for the statements, not a repitition.

- never comment out a code block without any comments.

- sync the logic with corresponding comments

  if the logic changes, change it's comment too

- keyword
  FIXME, TODO

- comments are placed directly above or directly right to the code block

- English comments are encouraged


best practice
#############

- package name should equal folder name

- add a blank line between switch's case statements

- use fmt.Sprintf instead of string concatenation

- never, ever trust players input


Unit Test
=========

- test file and tested file in the same folder

- filename ends with _test.go

  - e,g. all_test.go

- function name starts with 'Test' and has a param *testing.T

  - e,g. func TestAtomicString(t *testing.T) {}

- only test public interfaces

- sync between code and its unit test

- unit test readability is vital
  test code is a good documentation


Benchmark Test
==============

- test file and tested file in the same folder

- filename ends with _bench_test.go

  - e,g. table_bench_test.go

- function name starts with 'Benchmark' and has a param *testing.B

  - e,g. func BenchmarkIsSelectQuery(b *testing.B) {}


Request
=======

naming
######

lower case connected with underscore

::

    quest_id  // IS this form
    questId   // is NOT this form


Commit
======

- frequent comits is encouraged

  Commit as soon as your changes makes a logical unit

- be precise and exhaustive in your commit comments

- test code before you commit

- git diff before you commit

TERMS
=====

variables
#########

- channel

- cmd interface

- peer

- cluster


Tools
=====

gnu global
##########

::

    http://www.gnu.org/software/global/global.html

ack 
###

::

    http://beyondgrep.com/

