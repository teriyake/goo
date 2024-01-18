# Goo
A functional programming language inspired by and written in Go

### Main Design Principles

1. **Simplicity and Readability**: Goo prioritizes code clarity, making it accessible for beginners and efficient for experienced programmers. The language's syntax and structure aim for simplicity and readability.

2. **Functional Programming**: Goo emphasizes immutable data and treats functions as first-class citizens. It encourages the use of pure functions to ensure predictability and side-effect-free code.

3. **Strong Typing with Generics**: Implementing strong typing helps catch errors early. Generics allow for more flexible and reusable code without sacrificing type safety (coming soon...).

4. **Concurrency Support**: Inspired by Go, Goo offers support for concurrency (coming soon...).

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Syntax](#syntax-and-semantics-overview)
- [License](#license)

## Installation
Clone this repo:
```
git clone https://github.com/teriyake/goo.git
cd goo
```
Build from source (optional):
```
go build goo.go
```

## Usage
You can compile and run your `.goo` files with:
```
./goo path/to/src_code.goo
```
For more information on the cli flags available:
```
./goo -help
```

## Syntax and Semantics Overview

Goo adopts a Lisp-like syntax ;)

### Basic Structure
Programs in Goo consist of expressions enclosed in parentheses. For example, a simple "Hello, World!" program:

```
(print ('Hello, World!'))
```

### Variable Declaration
Variables are declared using the `let` keyword:

```
(let x:int 10)
```
Variables declared with `let` are immutable by default to encourage functional programming.  
Scope is lexical, with variables accessible within the block they are defined in and its sub-blocks.

### Function Definition
Functions are defined with `def`, and arguments are enclosed in parentheses:

```
(def add_x_y (x:int y:int)
  (ret (+ (x y)))
```
Functions are first-class citizens and can be passed around & manipulated like other data types.  
Eager evaluation is used, where function arguments are evaluated before the function call.

### Control Structures
Control structures are also enclosed in parentheses:

```
(if (> x 5)
  (print 'x is greater than 5')
  else (print 'x is not greater than 5'))
```
There are no `while` or `for` loops in Goo.

### Lambda Expressions
Lambda expressions are anonymous functions defined using `->`:

```
((x:int) -> (* x x))
```

### Map, Filter, Reduce
`map` takes a lambda expression and a list of arguments, and it returns a list of the same length as its arguments:
```
(map ((x:int) -> (* x 2)) (1 2 3 4 5))
;returns [2 4 6 8 10]
```
```
(map ((x:int) -> (if (> x 0) ('pos') else ('neg'))) (-1 2 -3))
; returns [neg pos neg]
```
`filter` selects elements from the arguments that satisfy the lambda expression. The list returned can have a different length than the arguments.
```
(filter ((x:int) -> (> x 0)) (-1 2 0))
; returns [2]
```
`reduce` combines the elements in a list into a single value by applying the lambda cumulatively from left to right. 
```
(reduce ((acc:int x:int) -> (operation on acc and x)) initial_value (list of elements))
```
```
(reduce ((acc:int x:int) -> (+ acc x)) 0 (1 2 3 4 5))
; returns 15
```

### Generics
coming soon...

### Error Handling
coming soon... 

### Comments
Comments start with a semicolon `;`:

```
; This is a comment
```

## Memory Management and Garbage Collection (Coming Soon...)
Goo employs automatic memory management with a generational garbage collector.


## Authors
- [Teri Ke](https://www.github.com/teriyake)

## License
[MIT](https://choosealicense.com/licenses/mit/)
