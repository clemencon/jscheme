// Abstract syntax (ECMA-262 1st edition, chapters 11-14).

use std::rc::Rc;

use crate::value::JsStr;

#[derive(Clone, Copy, Debug, PartialEq)]
pub enum BinOp {
    Mul,
    Div,
    Mod,
    Add,
    Sub,
    Shl,
    Sar,
    Shr,
    Lt,
    Gt,
    Le,
    Ge,
    Eq,
    Ne,
    BitAnd,
    BitXor,
    BitOr,
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub enum UnOp {
    Delete,
    Void,
    Typeof,
    PreInc,
    PreDec,
    PostInc,
    PostDec,
    Pos,
    Neg,
    BitNot,
    Not,
}

/// Marks an identifier that is not a local variable of the function it appears in.
pub const NO_SLOT: u32 = u32::MAX;

#[derive(Debug)]
pub enum Expr {
    This,
    /// An identifier, with its slot in the activation object when it names a local variable.
    Ident(JsStr, u32),
    Num(f64),
    Str(JsStr),
    Bool(bool),
    Null,
    Index(Box<Expr>, Box<Expr>),
    Dot(Box<Expr>, JsStr),
    Call(Box<Expr>, Vec<Expr>),
    New(Box<Expr>, Vec<Expr>),
    Unary(UnOp, Box<Expr>),
    Bin(BinOp, Box<Expr>, Box<Expr>),
    And(Box<Expr>, Box<Expr>),
    Or(Box<Expr>, Box<Expr>),
    Cond(Box<Expr>, Box<Expr>, Box<Expr>),
    Assign(Box<Expr>, Box<Expr>),
    OpAssign(BinOp, Box<Expr>, Box<Expr>),
    Comma(Box<Expr>, Box<Expr>),
}

#[derive(Debug)]
pub struct VarDecl {
    pub name: JsStr,
    pub slot: u32,
    pub init: Option<Expr>,
}

#[derive(Debug)]
pub enum ForInTarget {
    Var(VarDecl),
    Lhs(Expr),
}

#[derive(Debug)]
pub enum ForInit {
    Var(Vec<VarDecl>),
    Expr(Expr),
}

#[derive(Debug)]
pub enum Stmt {
    Block(Vec<Stmt>),
    Var(Vec<VarDecl>),
    Empty,
    Expr(Expr),
    If(Expr, Box<Stmt>, Option<Box<Stmt>>),
    While(Expr, Box<Stmt>),
    For(Option<ForInit>, Option<Expr>, Option<Expr>, Box<Stmt>),
    ForIn(ForInTarget, Expr, Box<Stmt>),
    Continue,
    Break,
    Return(Option<Expr>),
    With(Expr, Box<Stmt>),
}

#[derive(Debug)]
pub struct FuncDecl {
    pub name: JsStr,
    pub params: Vec<JsStr>,
    pub body: Vec<Stmt>,
    /// The names held in the activation object's slots: arguments, the parameters, the variables.
    pub locals: Vec<JsStr>,
    /// The slot of each parameter, by position.
    pub param_slots: Vec<u32>,
    /// Source text in FunctionDeclaration syntax, for Function.prototype.toString.
    pub source: String,
}

#[derive(Debug)]
pub struct Program {
    pub stmts: Vec<Stmt>,
    pub funcs: Vec<Rc<FuncDecl>>,
    pub vars: Vec<JsStr>,
}

/// The slots of a function's activation object: arguments first, then the parameters (a repeated
/// name shares one slot), then the variables. Returns the locals and the slot of each parameter.
pub fn layout_locals(arguments: &JsStr, params: &[JsStr], vars: &[JsStr]) -> (Vec<JsStr>, Vec<u32>) {
    let mut locals: Vec<JsStr> = vec![arguments.clone()];
    let slot_of = |locals: &mut Vec<JsStr>, n: &JsStr| -> u32 {
        match locals.iter().position(|l| l == n) {
            Some(i) => i as u32,
            None => {
                locals.push(n.clone());
                (locals.len() - 1) as u32
            }
        }
    };
    let param_slots = params.iter().map(|p| slot_of(&mut locals, p)).collect();
    for v in vars {
        slot_of(&mut locals, v);
    }
    (locals, param_slots)
}

/// Fills in the slot of every identifier of a function body that names one of its locals.
pub fn bind_slots(stmts: &mut [Stmt], locals: &[JsStr]) {
    let find = |n: &JsStr| locals.iter().position(|l| l == n).map(|i| i as u32).unwrap_or(NO_SLOT);
    for s in stmts {
        bind_stmt(s, &find);
    }
}

fn bind_var(d: &mut VarDecl, f: &dyn Fn(&JsStr) -> u32) {
    d.slot = f(&d.name);
    if let Some(e) = &mut d.init {
        bind_expr(e, f);
    }
}

fn bind_stmt(s: &mut Stmt, f: &dyn Fn(&JsStr) -> u32) {
    match s {
        Stmt::Block(v) => v.iter_mut().for_each(|s| bind_stmt(s, f)),
        Stmt::Var(v) => v.iter_mut().for_each(|d| bind_var(d, f)),
        Stmt::Empty | Stmt::Continue | Stmt::Break => {}
        Stmt::Expr(e) => bind_expr(e, f),
        Stmt::If(c, t, e) => {
            bind_expr(c, f);
            bind_stmt(t, f);
            if let Some(e) = e {
                bind_stmt(e, f);
            }
        }
        Stmt::While(c, b) => {
            bind_expr(c, f);
            bind_stmt(b, f);
        }
        Stmt::For(init, test, update, body) => {
            match init {
                Some(ForInit::Var(v)) => v.iter_mut().for_each(|d| bind_var(d, f)),
                Some(ForInit::Expr(e)) => bind_expr(e, f),
                None => {}
            }
            if let Some(e) = test {
                bind_expr(e, f);
            }
            if let Some(e) = update {
                bind_expr(e, f);
            }
            bind_stmt(body, f);
        }
        Stmt::ForIn(t, o, body) => {
            match t {
                ForInTarget::Var(d) => bind_var(d, f),
                ForInTarget::Lhs(e) => bind_expr(e, f),
            }
            bind_expr(o, f);
            bind_stmt(body, f);
        }
        Stmt::Return(e) => {
            if let Some(e) = e {
                bind_expr(e, f);
            }
        }
        Stmt::With(o, b) => {
            bind_expr(o, f);
            bind_stmt(b, f);
        }
    }
}

fn bind_expr(e: &mut Expr, f: &dyn Fn(&JsStr) -> u32) {
    match e {
        Expr::Ident(n, slot) => *slot = f(n),
        Expr::This | Expr::Num(_) | Expr::Str(_) | Expr::Bool(_) | Expr::Null => {}
        Expr::Dot(o, _) | Expr::Unary(_, o) => bind_expr(o, f),
        Expr::Index(a, b)
        | Expr::Bin(_, a, b)
        | Expr::And(a, b)
        | Expr::Or(a, b)
        | Expr::Assign(a, b)
        | Expr::OpAssign(_, a, b)
        | Expr::Comma(a, b) => {
            bind_expr(a, f);
            bind_expr(b, f);
        }
        Expr::Call(c, args) | Expr::New(c, args) => {
            bind_expr(c, f);
            args.iter_mut().for_each(|a| bind_expr(a, f));
        }
        Expr::Cond(a, b, c) => {
            bind_expr(a, f);
            bind_expr(b, f);
            bind_expr(c, f);
        }
    }
}
