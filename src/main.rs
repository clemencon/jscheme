// JSrust: an interpreter for ECMA-262, 1st edition.
// Host behaviour (invocation, exit codes, print, REPL) follows spec/host-contract.md.

mod ast;
mod builtins;
mod date;
mod interp;
mod lexer;
mod numconv;
mod parser;
mod value;

use std::io::{BufRead, IsTerminal, Write};

use interp::Interp;
use value::Value;

const STACK_SIZE: usize = 4 << 30;

fn usage(msg: &str) -> i32 {
    eprintln!("jsrust: {}", msg);
    eprintln!("usage: jsrust [<file.js> | -e <source>]");
    3
}

/// Runs a whole program: a syntax error stops it before it starts (2), a runtime error ends it (1).
fn run_source(src: &str) -> i32 {
    let prog = match parser::parse_program(src) {
        Ok(p) => p,
        Err(e) => {
            eprintln!("SyntaxError: {} (line {})", e.msg, e.line);
            return 2;
        }
    };
    let mut it = Interp::new();
    let r = it.run_program(&prog);
    it.flush();
    match r {
        Ok(_) => 0,
        Err(e) => {
            eprintln!("Uncaught {}", e.msg);
            1
        }
    }
}

fn repl() -> i32 {
    let stdin = std::io::stdin();
    let tty = stdin.is_terminal();
    let mut it = Interp::new();
    let mut buf = String::new();
    let mut input = stdin.lock();
    loop {
        if tty {
            let mut err = std::io::stderr();
            let _ = write!(err, "{}", if buf.is_empty() { "> " } else { "... " });
            let _ = err.flush();
        }
        let mut line = String::new();
        let eof = match input.read_line(&mut line) {
            Ok(0) => true,
            Ok(_) => false,
            Err(_) => true,
        };
        buf.push_str(&line);
        if eof && buf.trim().is_empty() {
            it.flush();
            return 0;
        }
        match parser::parse_program(&buf) {
            Err(e) if e.eof && !eof => continue,
            Err(e) => {
                it.flush();
                eprintln!("SyntaxError: {} (line {})", e.msg, e.line);
            }
            Ok(prog) => {
                match it.run_program(&prog) {
                    Ok(Some(v)) if !matches!(v, Value::Undef) => match it.to_string(v) {
                        Ok(s) => {
                            let line = format!("{}\n", value::rs(&s));
                            it.write_out(&line);
                        }
                        Err(e) => {
                            it.flush();
                            eprintln!("Uncaught {}", e.msg);
                            it.reset();
                        }
                    },
                    Ok(_) => {}
                    Err(e) => {
                        it.flush();
                        eprintln!("Uncaught {}", e.msg);
                        it.reset();
                    }
                }
                it.flush();
            }
        }
        buf.clear();
        if eof {
            it.flush();
            return 0;
        }
    }
}

fn run(args: Vec<String>) -> i32 {
    match args.len() {
        0 => repl(),
        1 if args[0] == "-e" => usage("-e needs a <source> argument"),
        1 if args[0].starts_with('-') && args[0] != "-" => usage(&format!("unknown option '{}'", args[0])),
        1 => match std::fs::read(&args[0]) {
            Ok(bytes) => run_source(&String::from_utf8_lossy(&bytes)),
            Err(e) => {
                eprintln!("jsrust: cannot read '{}': {}", args[0], e);
                3
            }
        },
        2 if args[0] == "-e" => run_source(&args[1]),
        _ => usage("too many arguments"),
    }
}

fn main() {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let handle = std::thread::Builder::new().stack_size(STACK_SIZE).spawn(move || run(args));
    let code = match handle {
        Ok(h) => h.join().unwrap_or(70),
        Err(e) => {
            eprintln!("jsrust: cannot start the interpreter thread: {}", e);
            70
        }
    };
    std::process::exit(code);
}
