// Execution: contexts (chapter 10), expressions (11), statements (12), functions (13), programs (14).

use std::io::Write;
use std::rc::Rc;

use crate::ast::*;
use crate::numconv::{num_to_string, str_to_number};
use crate::value::*;

pub struct JsErr {
    pub msg: String,
}

pub type JsResult<T> = Result<T, Box<JsErr>>;

pub fn throw<T>(msg: impl Into<String>) -> JsResult<T> {
    Err(Box::new(JsErr { msg: msg.into() }))
}

pub struct Scope {
    pub obj: ObjRef,
    pub next: Option<Rc<Scope>>,
}

#[derive(Clone)]
pub struct Ctx {
    pub scope: Rc<Scope>,
    pub var_obj: ObjRef,
    pub this: ObjRef,
}

#[derive(Clone, Copy, PartialEq, Debug)]
pub enum Ctl {
    Normal,
    Break,
    Continue,
    Return,
}

pub struct Comp {
    pub ctl: Ctl,
    pub val: Option<Value>,
}

impl Comp {
    fn normal() -> Comp {
        Comp { ctl: Ctl::Normal, val: None }
    }
}

#[derive(Clone)]
enum Ref {
    Val(Value),
    Prop(Value, JsStr),
    Unres(JsStr),
}

#[derive(Clone, Copy, PartialEq)]
pub enum Hint {
    Number,
    String,
}

/// Interned names and strings used by the interpreter.
pub struct Names {
    pub length: JsStr,
    pub prototype: JsStr,
    pub constructor: JsStr,
    pub arguments: JsStr,
    pub callee: JsStr,
    pub to_string: JsStr,
    pub value_of: JsStr,
    pub undefined: JsStr,
    pub null: JsStr,
    pub true_: JsStr,
    pub false_: JsStr,
    pub object: JsStr,
    pub boolean: JsStr,
    pub number: JsStr,
    pub string: JsStr,
    pub function: JsStr,
    pub empty: JsStr,
}

pub const MAX_DEPTH: usize = 400_000;

pub struct Interp {
    pub global: ObjRef,
    pub global_scope: Rc<Scope>,
    pub object_proto: ObjRef,
    pub function_proto: ObjRef,
    pub array_proto: ObjRef,
    pub string_proto: ObjRef,
    pub boolean_proto: ObjRef,
    pub number_proto: ObjRef,
    pub date_proto: ObjRef,
    pub ctx: Ctx,
    pub depth: usize,
    pub n: Names,
    pub out: std::io::BufWriter<std::io::Stdout>,
    pub flush_each: bool,
    pub rng: u64,
    index_keys: Vec<JsStr>,
}

pub fn to_boolean(v: &Value) -> bool {
    match v {
        Value::Undef | Value::Null => false,
        Value::Bool(b) => *b,
        Value::Num(n) => !(*n == 0.0 || n.is_nan()),
        Value::Str(s) => !s.is_empty(),
        Value::Obj(_) => true,
    }
}

pub fn to_integer(n: f64) -> f64 {
    if n.is_nan() {
        0.0
    } else if n == 0.0 || n.is_infinite() {
        n
    } else {
        n.trunc()
    }
}

pub fn to_uint32(n: f64) -> u32 {
    if !n.is_finite() {
        return 0;
    }
    n.trunc().rem_euclid(4294967296.0) as u64 as u32
}

pub fn to_int32(n: f64) -> i32 {
    to_uint32(n) as i32
}

pub fn to_uint16(n: f64) -> u16 {
    if !n.is_finite() {
        return 0;
    }
    n.trunc().rem_euclid(65536.0) as u32 as u16
}

pub fn concat(a: &[u16], b: &[u16]) -> JsStr {
    let mut v = Vec::with_capacity(a.len() + b.len());
    v.extend_from_slice(a);
    v.extend_from_slice(b);
    v.into()
}

pub fn arg(args: &[Value], i: usize) -> Value {
    args.get(i).cloned().unwrap_or(Value::Undef)
}

fn is_activation(o: &ObjRef) -> bool {
    matches!(o.borrow().kind, Kind::Activation)
}

impl Interp {
    pub fn new() -> Interp {
        let object_proto = Object::new("Object", None, Kind::Plain);
        let function_proto = Object::new(
            "Function",
            Some(object_proto.clone()),
            Kind::Func(Func::Native(Native { name: "", call: |_, _, _| Ok(Value::Undef), ctor: None })),
        );
        let mk = |class: &'static str, kind: Kind| Object::new(class, Some(object_proto.clone()), kind);
        let array_proto = mk("Array", Kind::Array);
        let string_proto = mk("String", Kind::Str(js("")));
        let boolean_proto = mk("Boolean", Kind::Bool(false));
        let number_proto = mk("Number", Kind::Num(0.0));
        let date_proto = mk("Date", Kind::Date(f64::NAN));
        let global = mk("Object", Kind::Plain);
        let global_scope = Rc::new(Scope { obj: global.clone(), next: None });
        let n = Names {
            length: intern_str("length"),
            prototype: intern_str("prototype"),
            constructor: intern_str("constructor"),
            arguments: intern_str("arguments"),
            callee: intern_str("callee"),
            to_string: intern_str("toString"),
            value_of: intern_str("valueOf"),
            undefined: intern_str("undefined"),
            null: intern_str("null"),
            true_: intern_str("true"),
            false_: intern_str("false"),
            object: intern_str("object"),
            boolean: intern_str("boolean"),
            number: intern_str("number"),
            string: intern_str("string"),
            function: intern_str("function"),
            empty: intern_str(""),
        };
        use std::io::IsTerminal;
        let seed = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map(|d| d.as_nanos() as u64)
            .unwrap_or(0x9E3779B97F4A7C15);
        let mut it = Interp {
            ctx: Ctx { scope: global_scope.clone(), var_obj: global.clone(), this: global.clone() },
            global,
            global_scope,
            object_proto,
            function_proto,
            array_proto,
            string_proto,
            boolean_proto,
            number_proto,
            date_proto,
            depth: 0,
            n,
            out: std::io::BufWriter::with_capacity(1 << 16, std::io::stdout()),
            flush_each: std::io::stdout().is_terminal(),
            rng: seed | 1,
            index_keys: Vec::new(),
        };
        it.array_proto.borrow_mut().set(it.n.length.clone(), Value::Num(0.0), DE | DD);
        it.string_proto.borrow_mut().set(it.n.length.clone(), Value::Num(0.0), RO | DE | DD);
        crate::builtins::setup(&mut it);
        it
    }

    /// Resets the execution context to global code (after an error in the REPL).
    pub fn reset(&mut self) {
        self.ctx = Ctx { scope: self.global_scope.clone(), var_obj: self.global.clone(), this: self.global.clone() };
        self.depth = 0;
    }

    pub fn write_out(&mut self, s: &str) {
        let _ = self.out.write_all(s.as_bytes());
        if self.flush_each {
            let _ = self.out.flush();
        }
    }

    pub fn flush(&mut self) {
        let _ = self.out.flush();
    }

    // Keys.

    pub fn index_key(&mut self, i: usize) -> JsStr {
        if i < 1 << 16 {
            while self.index_keys.len() <= i {
                let k = intern_str(&self.index_keys.len().to_string());
                self.index_keys.push(k);
            }
            return self.index_keys[i].clone();
        }
        js(&i.to_string())
    }

    pub fn num_key(&mut self, n: f64) -> JsStr {
        if n >= 0.0 && n < 4294967295.0 && n.fract() == 0.0 {
            return self.index_key(n as usize);
        }
        js(&num_to_string(n))
    }

    pub fn to_key(&mut self, v: Value) -> JsResult<JsStr> {
        match v {
            Value::Str(s) => Ok(s),
            Value::Num(n) => Ok(self.num_key(n)),
            v => self.to_string(v),
        }
    }

    // Type conversion (chapter 9).

    pub fn to_primitive(&mut self, v: Value, hint: Option<Hint>) -> JsResult<Value> {
        match v {
            Value::Obj(o) => self.default_value(&o, hint),
            v => Ok(v),
        }
    }

    /// [[DefaultValue]] (§8.6.2.6).
    pub fn default_value(&mut self, o: &ObjRef, hint: Option<Hint>) -> JsResult<Value> {
        let hint = hint.unwrap_or_else(|| if o.borrow().class == "Date" { Hint::String } else { Hint::Number });
        let order = if hint == Hint::String {
            [self.n.to_string.clone(), self.n.value_of.clone()]
        } else {
            [self.n.value_of.clone(), self.n.to_string.clone()]
        };
        for name in order {
            let f = get(o, &name);
            if f.is_obj() {
                let r = self.call(&f, Value::Obj(o.clone()), &[])?;
                if !r.is_obj() {
                    return Ok(r);
                }
            }
        }
        throw("TypeError: cannot convert object to primitive value")
    }

    pub fn to_number(&mut self, v: Value) -> JsResult<f64> {
        Ok(match v {
            Value::Num(n) => n,
            Value::Undef => f64::NAN,
            Value::Null => 0.0,
            Value::Bool(b) => b as u8 as f64,
            Value::Str(s) => str_to_number(&s),
            Value::Obj(o) => {
                let p = self.default_value(&o, Some(Hint::Number))?;
                return self.to_number(p);
            }
        })
    }

    pub fn to_string(&mut self, v: Value) -> JsResult<JsStr> {
        Ok(match v {
            Value::Str(s) => s,
            Value::Num(n) => js(&num_to_string(n)),
            Value::Undef => self.n.undefined.clone(),
            Value::Null => self.n.null.clone(),
            Value::Bool(true) => self.n.true_.clone(),
            Value::Bool(false) => self.n.false_.clone(),
            Value::Obj(o) => {
                let p = self.default_value(&o, Some(Hint::String))?;
                return self.to_string(p);
            }
        })
    }

    pub fn to_object(&mut self, v: Value) -> JsResult<ObjRef> {
        Ok(match v {
            Value::Obj(o) => o,
            Value::Undef | Value::Null => return throw("TypeError: cannot convert undefined or null to an object"),
            Value::Bool(b) => Object::new("Boolean", Some(self.boolean_proto.clone()), Kind::Bool(b)),
            Value::Num(n) => Object::new("Number", Some(self.number_proto.clone()), Kind::Num(n)),
            Value::Str(s) => self.new_string_object(s),
        })
    }

    pub fn new_string_object(&mut self, s: JsStr) -> ObjRef {
        let len = s.len() as f64;
        let o = Object::new("String", Some(self.string_proto.clone()), Kind::Str(s));
        o.borrow_mut().set(self.n.length.clone(), Value::Num(len), RO | DE | DD);
        o
    }

    pub fn new_object(&mut self) -> ObjRef {
        Object::new("Object", Some(self.object_proto.clone()), Kind::Plain)
    }

    pub fn new_array(&mut self) -> ObjRef {
        let a = Object::new("Array", Some(self.array_proto.clone()), Kind::Array);
        a.borrow_mut().set(self.n.length.clone(), Value::Num(0.0), DE | DD);
        a
    }

    // Property access.

    fn check_coercible(&self, b: &Value) -> JsResult<()> {
        match b {
            Value::Undef => throw("TypeError: cannot read a property of undefined"),
            Value::Null => throw("TypeError: cannot read a property of null"),
            _ => Ok(()),
        }
    }

    /// GetValue of a reference whose base is the value b (ToObject applied implicitly).
    pub fn get_prop_of(&mut self, b: &Value, name: &JsStr) -> JsResult<Value> {
        match b {
            Value::Obj(o) => Ok(get(o, name)),
            Value::Str(s) => {
                if **name == *self.n.length {
                    Ok(Value::Num(s.len() as f64))
                } else {
                    Ok(get(&self.string_proto, name))
                }
            }
            Value::Num(_) => Ok(get(&self.number_proto, name)),
            Value::Bool(_) => Ok(get(&self.boolean_proto, name)),
            _ => {
                self.check_coercible(b)?;
                Ok(Value::Undef)
            }
        }
    }

    /// [[Put]], with the Array variation (§15.4.5.1).
    pub fn put(&mut self, o: &ObjRef, k: &JsStr, v: Value) -> JsResult<()> {
        if !matches!(o.borrow().kind, Kind::Array) {
            put_plain(o, k, v);
            return Ok(());
        }
        if !can_put(o, k) {
            return Ok(());
        }
        if **k == *self.n.length {
            let new_len = to_uint32(self.to_number(v)?) as f64;
            let mut b = o.borrow_mut();
            let old_len = match b.props.get(k) {
                Some(Prop { val: Value::Num(n), .. }) => *n,
                _ => 0.0,
            };
            if new_len < old_len {
                let doomed: Vec<JsStr> = b
                    .props
                    .iter()
                    .filter(|(name, p)| {
                        p.attrs & DD == 0 && matches!(array_index(name), Some(i) if (i as f64) >= new_len && (i as f64) < old_len)
                    })
                    .map(|(name, _)| name.clone())
                    .collect();
                for d in doomed {
                    b.props.remove(&d);
                }
            }
            match b.props.get_mut(k) {
                Some(p) => p.val = Value::Num(new_len),
                None => b.set(k.clone(), Value::Num(new_len), DE | DD),
            }
            return Ok(());
        }
        let mut b = o.borrow_mut();
        match b.props.get_mut(k) {
            Some(p) => p.val = v,
            None => b.props.insert(k.clone(), Prop { val: v, attrs: 0 }),
        }
        if let Some(i) = array_index(k) {
            let len = match b.props.get(&self.n.length) {
                Some(Prop { val: Value::Num(n), .. }) => Some(*n),
                _ => None,
            };
            match len {
                Some(l) if (i as f64) < l => {}
                Some(_) => b.props.get_mut(&self.n.length).unwrap().val = Value::Num(i as f64 + 1.0),
                None => b.set(self.n.length.clone(), Value::Num(i as f64 + 1.0), DE | DD),
            }
        }
        Ok(())
    }

    // References (§8.7).

    fn resolve(&self, n: &[u16]) -> Option<ObjRef> {
        let mut s = &self.ctx.scope;
        loop {
            if has_property(&s.obj, n) {
                return Some(s.obj.clone());
            }
            match &s.next {
                Some(nx) => s = nx,
                None => return None,
            }
        }
    }

    /// The activation object, when identifier slot `slot` can be read from it directly: the
    /// code runs in a function's own context and no with statement has added a scope.
    #[inline]
    fn slot_scope(&self, slot: u32) -> Option<&ObjRef> {
        if slot != NO_SLOT && Rc::ptr_eq(&self.ctx.scope.obj, &self.ctx.var_obj) { Some(&self.ctx.var_obj) } else { None }
    }

    /// Resolves an identifier, taking its slot when it has one and the slot still holds it.
    #[inline]
    fn resolve_slot(&self, n: &JsStr, slot: u32) -> Option<ObjRef> {
        if let Some(o) = self.slot_scope(slot) {
            if o.borrow().props.get_at(slot as usize, n).is_some() {
                return Some(o.clone());
            }
        }
        self.resolve(n)
    }

    #[inline]
    fn get_ident_slot(&self, n: &JsStr, slot: u32) -> JsResult<Value> {
        if let Some(o) = self.slot_scope(slot) {
            if let Some(p) = o.borrow().props.get_at(slot as usize, n) {
                return Ok(p.val.clone());
            }
        }
        self.get_ident(n)
    }

    /// Assigns to a resolved identifier, writing its slot directly when possible.
    fn assign_slot(&mut self, n: &JsStr, slot: u32, base: Option<ObjRef>, v: Value) -> JsResult<()> {
        if let Some(o) = &base {
            if slot != NO_SLOT && Rc::ptr_eq(o, &self.ctx.var_obj) {
                if let Some(p) = o.borrow_mut().props.get_at_mut(slot as usize, n) {
                    if p.attrs & RO == 0 {
                        p.val = v;
                        return Ok(());
                    }
                }
            }
        }
        self.assign_ident(n, base, v)
    }

    fn get_ident(&self, n: &JsStr) -> JsResult<Value> {
        let mut s = &self.ctx.scope;
        loop {
            if let Some(v) = lookup(&s.obj, n) {
                return Ok(v);
            }
            match &s.next {
                Some(nx) => s = nx,
                None => return throw(format!("ReferenceError: {} is not defined", rs(n))),
            }
        }
    }

    fn eval_ref(&mut self, e: &Expr) -> JsResult<Ref> {
        Ok(match e {
            Expr::Ident(n, slot) => match self.resolve_slot(n, *slot) {
                Some(o) => Ref::Prop(Value::Obj(o), n.clone()),
                None => Ref::Unres(n.clone()),
            },
            Expr::Dot(o, n) => {
                let b = self.eval(o)?;
                self.check_coercible(&b)?;
                Ref::Prop(b, n.clone())
            }
            Expr::Index(o, k) => {
                let b = self.eval(o)?;
                let kv = self.eval(k)?;
                self.check_coercible(&b)?;
                let key = self.to_key(kv)?;
                Ref::Prop(b, key)
            }
            e => Ref::Val(self.eval(e)?),
        })
    }

    fn get_value(&mut self, r: &Ref) -> JsResult<Value> {
        match r {
            Ref::Val(v) => Ok(v.clone()),
            Ref::Prop(b, n) => self.get_prop_of(b, n),
            Ref::Unres(n) => throw(format!("ReferenceError: {} is not defined", rs(n))),
        }
    }

    fn put_value(&mut self, r: Ref, v: Value) -> JsResult<()> {
        match r {
            Ref::Val(_) => throw("ReferenceError: invalid assignment target"),
            Ref::Prop(Value::Obj(o), n) => self.put(&o, &n, v),
            Ref::Prop(_, _) => Ok(()),
            Ref::Unres(n) => {
                let g = self.global.clone();
                self.put(&g, &n, v)
            }
        }
    }

    // Function calls (§10.2.3, §11.2.3, §13).

    pub fn call(&mut self, f: &Value, this: Value, args: &[Value]) -> JsResult<Value> {
        let fo = match f {
            Value::Obj(o) => o,
            _ => return throw("TypeError: value is not a function"),
        };
        enum K {
            User(Rc<FuncDecl>),
            Native(NativeFn),
        }
        let k = match &fo.borrow().kind {
            Kind::Func(Func::User(d)) => K::User(d.clone()),
            Kind::Func(Func::Native(n)) => K::Native(n.call),
            _ => return throw("TypeError: object is not a function"),
        };
        match k {
            K::User(d) => self.call_user(fo, &d, this, args),
            K::Native(nf) => {
                let this = match this {
                    Value::Undef | Value::Null => Value::Obj(self.global.clone()),
                    t => t,
                };
                nf(self, this, args)
            }
        }
    }

    pub fn construct(&mut self, f: &Value, args: &[Value]) -> JsResult<Value> {
        let fo = match f {
            Value::Obj(o) => o,
            _ => return throw("TypeError: value is not a constructor"),
        };
        enum K {
            User(Rc<FuncDecl>),
            Native(NativeFn),
        }
        let k = match &fo.borrow().kind {
            Kind::Func(Func::User(d)) => K::User(d.clone()),
            Kind::Func(Func::Native(Native { ctor: Some(c), .. })) => K::Native(*c),
            _ => return throw("TypeError: object is not a constructor"),
        };
        match k {
            K::User(d) => {
                let proto = match get(fo, &self.n.prototype) {
                    Value::Obj(p) => p,
                    _ => self.object_proto.clone(),
                };
                let o = Object::new("Object", Some(proto), Kind::Plain);
                let r = self.call_user(fo, &d, Value::Obj(o.clone()), args)?;
                Ok(if r.is_obj() { r } else { Value::Obj(o) })
            }
            K::Native(c) => c(self, Value::Undef, args),
        }
    }

    fn call_user(&mut self, f: &ObjRef, d: &Rc<FuncDecl>, this: Value, args: &[Value]) -> JsResult<Value> {
        if self.depth >= MAX_DEPTH {
            return throw("RangeError: too much recursion");
        }
        let this = match this {
            Value::Obj(o) => o,
            Value::Undef | Value::Null => self.global.clone(),
            v => self.to_object(v)?,
        };
        let act = Object::new("Object", None, Kind::Activation);
        let np = d.params.len();
        let map: Vec<Option<JsStr>> = (0..args.len().min(np))
            .map(|i| {
                let p = &d.params[i];
                if d.params[i + 1..].iter().any(|q| q == p) { None } else { Some(p.clone()) }
            })
            .collect();
        let argo = Object::new("Object", Some(self.object_proto.clone()), Kind::Arguments { act: Rc::downgrade(&act), map });
        {
            let mut a = argo.borrow_mut();
            a.set(self.n.callee.clone(), Value::Obj(f.clone()), DE);
            a.set(self.n.length.clone(), Value::Num(args.len() as f64), DE);
        }
        for (i, v) in args.iter().enumerate() {
            let k = self.index_key(i);
            argo.borrow_mut().set(k, v.clone(), DE);
        }
        {
            // The locals go in slot order (see ast::layout_locals): arguments, the parameters
            // (a later parameter of the same name takes its value), then the variables.
            let mut vals = vec![Value::Undef; d.locals.len()];
            vals[0] = Value::Obj(argo.clone());
            for (i, &slot) in d.param_slots.iter().enumerate() {
                vals[slot as usize] = args.get(i).cloned().unwrap_or(Value::Undef);
            }
            let mut ab = act.borrow_mut();
            for (k, v) in d.locals.iter().zip(vals) {
                ab.props.insert(k.clone(), Prop { val: v, attrs: DD });
            }
        }
        let old = {
            let mut fb = f.borrow_mut();
            let old = fb.props.get(&self.n.arguments).cloned();
            let attrs = old.as_ref().map(|p| p.attrs).unwrap_or(DE);
            fb.set(self.n.arguments.clone(), Value::Obj(argo), attrs);
            old
        };
        let ctx = Ctx {
            scope: Rc::new(Scope { obj: act.clone(), next: Some(self.global_scope.clone()) }),
            var_obj: act,
            this,
        };
        let saved = std::mem::replace(&mut self.ctx, ctx);
        self.depth += 1;
        let r = self.exec_list(&d.body);
        self.depth -= 1;
        self.ctx = saved;
        {
            let mut fb = f.borrow_mut();
            match old {
                Some(p) => fb.props.insert(self.n.arguments.clone(), p),
                None => {
                    fb.props.remove(&self.n.arguments);
                }
            }
        }
        let c = r?;
        Ok(match c.ctl {
            Ctl::Return => c.val.unwrap_or(Value::Undef),
            _ => Value::Undef,
        })
    }

    pub fn make_function(&mut self, d: Rc<FuncDecl>) -> ObjRef {
        let np = d.params.len() as f64;
        let f = Object::new("Function", Some(self.function_proto.clone()), Kind::Func(Func::User(d)));
        let proto = self.new_object();
        proto.borrow_mut().set(self.n.constructor.clone(), Value::Obj(f.clone()), DE);
        {
            let mut b = f.borrow_mut();
            b.set(self.n.length.clone(), Value::Num(np), RO | DD | DE);
            b.set(self.n.prototype.clone(), Value::Obj(proto), DE);
            b.set(self.n.arguments.clone(), Value::Null, DE);
        }
        f
    }

    // Programs (chapter 14) and variable instantiation (§10.1.3).

    /// Runs a program in the current context (global code, or eval code).
    pub fn run_program(&mut self, prog: &Program) -> JsResult<Option<Value>> {
        let var_obj = self.ctx.var_obj.clone();
        for f in &prog.funcs {
            let fo = self.make_function(f.clone());
            var_obj.borrow_mut().set(f.name.clone(), Value::Obj(fo), 0);
        }
        for v in &prog.vars {
            if !has_property(&var_obj, v) {
                var_obj.borrow_mut().set(v.clone(), Value::Undef, 0);
            }
        }
        let c = self.exec_list(&prog.stmts)?;
        Ok(c.val)
    }

    // Statements (chapter 12).

    fn exec_list(&mut self, stmts: &[Stmt]) -> JsResult<Comp> {
        let mut val: Option<Value> = None;
        for s in stmts {
            let r = self.exec(s)?;
            if r.ctl != Ctl::Normal {
                return Ok(Comp { ctl: r.ctl, val: r.val.or(val) });
            }
            if r.val.is_some() {
                val = r.val;
            }
        }
        Ok(Comp { ctl: Ctl::Normal, val })
    }

    fn assign_ident(&mut self, n: &JsStr, base: Option<ObjRef>, v: Value) -> JsResult<()> {
        match base {
            Some(o) => self.put(&o, n, v),
            None => {
                let g = self.global.clone();
                self.put(&g, n, v)
            }
        }
    }

    fn exec_vars(&mut self, decls: &[VarDecl]) -> JsResult<()> {
        for d in decls {
            if let Some(e) = &d.init {
                let base = self.resolve_slot(&d.name, d.slot);
                let v = self.eval(e)?;
                self.assign_slot(&d.name, d.slot, base, v)?;
            }
        }
        Ok(())
    }

    fn exec(&mut self, s: &Stmt) -> JsResult<Comp> {
        match s {
            Stmt::Expr(e) => {
                let v = self.eval(e)?;
                Ok(Comp { ctl: Ctl::Normal, val: Some(v) })
            }
            Stmt::Block(v) => self.exec_list(v),
            Stmt::Var(decls) => {
                self.exec_vars(decls)?;
                Ok(Comp::normal())
            }
            Stmt::Empty => Ok(Comp::normal()),
            Stmt::If(c, t, e) => {
                let cv = self.eval(c)?;
                if to_boolean(&cv) {
                    self.exec(t)
                } else if let Some(e) = e {
                    self.exec(e)
                } else {
                    Ok(Comp::normal())
                }
            }
            Stmt::While(c, body) => {
                let mut val = None;
                loop {
                    let cv = self.eval(c)?;
                    if !to_boolean(&cv) {
                        break;
                    }
                    let r = self.exec(body)?;
                    if r.ctl == Ctl::Return {
                        return Ok(r);
                    }
                    if r.val.is_some() {
                        val = r.val;
                    }
                    if r.ctl == Ctl::Break {
                        break;
                    }
                }
                Ok(Comp { ctl: Ctl::Normal, val })
            }
            Stmt::For(init, test, update, body) => {
                match init {
                    Some(ForInit::Var(d)) => self.exec_vars(d)?,
                    Some(ForInit::Expr(e)) => {
                        self.eval(e)?;
                    }
                    None => {}
                }
                let mut val = None;
                loop {
                    if let Some(t) = test {
                        let cv = self.eval(t)?;
                        if !to_boolean(&cv) {
                            break;
                        }
                    }
                    let r = self.exec(body)?;
                    if r.ctl == Ctl::Return {
                        return Ok(r);
                    }
                    if r.val.is_some() {
                        val = r.val;
                    }
                    if r.ctl == Ctl::Break {
                        break;
                    }
                    if let Some(u) = update {
                        self.eval(u)?;
                    }
                }
                Ok(Comp { ctl: Ctl::Normal, val })
            }
            Stmt::ForIn(target, obj, body) => self.exec_for_in(target, obj, body),
            Stmt::Continue => Ok(Comp { ctl: Ctl::Continue, val: None }),
            Stmt::Break => Ok(Comp { ctl: Ctl::Break, val: None }),
            Stmt::Return(e) => {
                let v = match e {
                    Some(e) => self.eval(e)?,
                    None => Value::Undef,
                };
                Ok(Comp { ctl: Ctl::Return, val: Some(v) })
            }
            Stmt::With(e, body) => {
                let v = self.eval(e)?;
                let o = self.to_object(v)?;
                let saved = self.ctx.scope.clone();
                self.ctx.scope = Rc::new(Scope { obj: o, next: Some(saved.clone()) });
                let r = self.exec(body);
                self.ctx.scope = saved;
                r
            }
        }
    }

    fn exec_for_in(&mut self, target: &ForInTarget, obj: &Expr, body: &Stmt) -> JsResult<Comp> {
        if let ForInTarget::Var(VarDecl { name, slot, init: Some(init) }) = target {
            let base = self.resolve_slot(name, *slot);
            let v = self.eval(init)?;
            self.assign_slot(name, *slot, base, v)?;
        }
        let ov = self.eval(obj)?;
        let o = self.to_object(ov)?;
        // Collect the enumerable names along the prototype chain; shadowed names appear once.
        let mut seen: std::collections::HashSet<JsStr, Fx> = Default::default();
        let mut names = Vec::new();
        let mut cur = Some(o.clone());
        while let Some(c) = cur {
            let b = c.borrow();
            for (k, p) in b.props.iter() {
                if seen.insert(k.clone()) && p.attrs & DE == 0 {
                    names.push(k.clone());
                }
            }
            cur = b.proto.clone();
        }
        let mut val = None;
        for name in names {
            if !has_property(&o, &name) {
                continue;
            }
            let r = match target {
                ForInTarget::Var(d) => match self.resolve_slot(&d.name, d.slot) {
                    Some(b) => Ref::Prop(Value::Obj(b), d.name.clone()),
                    None => Ref::Unres(d.name.clone()),
                },
                ForInTarget::Lhs(e) => self.eval_ref(e)?,
            };
            self.put_value(r, Value::Str(name))?;
            let r = self.exec(body)?;
            if r.ctl == Ctl::Return {
                return Ok(r);
            }
            if r.val.is_some() {
                val = r.val;
            }
            if r.ctl == Ctl::Break {
                break;
            }
        }
        Ok(Comp { ctl: Ctl::Normal, val })
    }

    // Expressions (chapter 11).

    fn eval_args(&mut self, args: &[Expr]) -> JsResult<Vec<Value>> {
        let mut v = Vec::with_capacity(args.len());
        for a in args {
            v.push(self.eval(a)?);
        }
        Ok(v)
    }

    pub fn eval(&mut self, e: &Expr) -> JsResult<Value> {
        match e {
            Expr::Num(n) => Ok(Value::Num(*n)),
            Expr::Ident(n, slot) => self.get_ident_slot(n, *slot),
            Expr::Str(s) => Ok(Value::Str(s.clone())),
            Expr::Bool(b) => Ok(Value::Bool(*b)),
            Expr::Null => Ok(Value::Null),
            Expr::This => Ok(Value::Obj(self.ctx.this.clone())),
            Expr::Dot(o, n) => {
                let b = self.eval(o)?;
                self.get_prop_of(&b, n)
            }
            Expr::Index(o, k) => {
                let b = self.eval(o)?;
                let kv = self.eval(k)?;
                self.check_coercible(&b)?;
                if let (Value::Obj(ob), Value::Num(n)) = (&b, &kv) {
                    if *n >= 0.0 && *n < 65536.0 && n.fract() == 0.0 {
                        let key = self.index_key(*n as usize);
                        return Ok(get(ob, &key));
                    }
                }
                let key = self.to_key(kv)?;
                self.get_prop_of(&b, &key)
            }
            Expr::Call(callee, args) => self.eval_call(callee, args),
            Expr::New(c, args) => {
                let f = self.eval(c)?;
                let argv = self.eval_args(args)?;
                self.construct(&f, &argv)
            }
            Expr::Bin(op, l, r) => {
                let a = self.eval(l)?;
                let b = self.eval(r)?;
                self.binop(*op, a, b)
            }
            Expr::Unary(op, x) => self.eval_unary(*op, x),
            Expr::And(l, r) => {
                let a = self.eval(l)?;
                if !to_boolean(&a) {
                    return Ok(a);
                }
                self.eval(r)
            }
            Expr::Or(l, r) => {
                let a = self.eval(l)?;
                if to_boolean(&a) {
                    return Ok(a);
                }
                self.eval(r)
            }
            Expr::Cond(c, a, b) => {
                let cv = self.eval(c)?;
                if to_boolean(&cv) { self.eval(a) } else { self.eval(b) }
            }
            Expr::Assign(l, r) => {
                if let Expr::Ident(n, slot) = &**l {
                    let base = self.resolve_slot(n, *slot);
                    let v = self.eval(r)?;
                    self.assign_slot(n, *slot, base, v.clone())?;
                    return Ok(v);
                }
                let rf = self.eval_ref(l)?;
                let v = self.eval(r)?;
                self.put_value(rf, v.clone())?;
                Ok(v)
            }
            Expr::OpAssign(op, l, r) => {
                let rf = self.eval_ref(l)?;
                let a = self.get_value(&rf)?;
                let b = self.eval(r)?;
                let v = self.binop(*op, a, b)?;
                self.put_value(rf, v.clone())?;
                Ok(v)
            }
            Expr::Comma(l, r) => {
                self.eval(l)?;
                self.eval(r)
            }
        }
    }

    fn eval_call(&mut self, callee: &Expr, args: &[Expr]) -> JsResult<Value> {
        match callee {
            Expr::Ident(n, slot) => {
                let base = self.resolve_slot(n, *slot);
                let argv = self.eval_args(args)?;
                match base {
                    Some(o) => {
                        let f = get(&o, n);
                        let this = if is_activation(&o) { Value::Null } else { Value::Obj(o) };
                        self.call(&f, this, &argv)
                    }
                    None => throw(format!("ReferenceError: {} is not defined", rs(n))),
                }
            }
            Expr::Dot(_, _) | Expr::Index(_, _) => {
                let r = self.eval_ref(callee)?;
                let argv = self.eval_args(args)?;
                let f = self.get_value(&r)?;
                let this = match r {
                    Ref::Prop(b, _) => match &b {
                        Value::Obj(o) if is_activation(o) => Value::Null,
                        _ => b,
                    },
                    _ => Value::Null,
                };
                self.call(&f, this, &argv)
            }
            _ => {
                let f = self.eval(callee)?;
                let argv = self.eval_args(args)?;
                self.call(&f, Value::Null, &argv)
            }
        }
    }

    pub fn type_of(&self, v: &Value) -> JsStr {
        match v {
            Value::Undef => self.n.undefined.clone(),
            Value::Null => self.n.object.clone(),
            Value::Bool(_) => self.n.boolean.clone(),
            Value::Num(_) => self.n.number.clone(),
            Value::Str(_) => self.n.string.clone(),
            Value::Obj(o) => {
                if o.borrow().is_callable() {
                    self.n.function.clone()
                } else {
                    self.n.object.clone()
                }
            }
        }
    }

    fn eval_unary(&mut self, op: UnOp, x: &Expr) -> JsResult<Value> {
        match op {
            UnOp::Delete => {
                let r = self.eval_ref(x)?;
                match r {
                    Ref::Val(_) => throw("ReferenceError: delete of a value that is not a reference"),
                    Ref::Unres(_) => Ok(Value::Bool(true)),
                    Ref::Prop(Value::Obj(o), n) => Ok(Value::Bool(delete(&o, &n))),
                    Ref::Prop(b, n) => {
                        let o = self.to_object(b)?;
                        Ok(Value::Bool(delete(&o, &n)))
                    }
                }
            }
            UnOp::Void => {
                self.eval(x)?;
                Ok(Value::Undef)
            }
            UnOp::Typeof => {
                let r = self.eval_ref(x)?;
                if let Ref::Unres(_) = r {
                    return Ok(Value::Str(self.n.undefined.clone()));
                }
                let v = self.get_value(&r)?;
                Ok(Value::Str(self.type_of(&v)))
            }
            UnOp::PreInc | UnOp::PreDec | UnOp::PostInc | UnOp::PostDec => {
                let r = self.eval_ref(x)?;
                let old = self.get_value(&r)?;
                let n = self.to_number(old)?;
                let nv = if matches!(op, UnOp::PreInc | UnOp::PostInc) { n + 1.0 } else { n - 1.0 };
                self.put_value(r, Value::Num(nv))?;
                Ok(Value::Num(if matches!(op, UnOp::PreInc | UnOp::PreDec) { nv } else { n }))
            }
            UnOp::Pos => {
                let v = self.eval(x)?;
                Ok(Value::Num(self.to_number(v)?))
            }
            UnOp::Neg => {
                let v = self.eval(x)?;
                Ok(Value::Num(-self.to_number(v)?))
            }
            UnOp::BitNot => {
                let v = self.eval(x)?;
                let n = self.to_number(v)?;
                Ok(Value::Num(!to_int32(n) as f64))
            }
            UnOp::Not => {
                let v = self.eval(x)?;
                Ok(Value::Bool(!to_boolean(&v)))
            }
        }
    }

    pub fn binop(&mut self, op: BinOp, a: Value, b: Value) -> JsResult<Value> {
        if let (Value::Num(x), Value::Num(y)) = (&a, &b) {
            let (x, y) = (*x, *y);
            return Ok(match op {
                BinOp::Add => Value::Num(x + y),
                BinOp::Sub => Value::Num(x - y),
                BinOp::Mul => Value::Num(x * y),
                BinOp::Div => Value::Num(x / y),
                BinOp::Mod => Value::Num(x % y),
                BinOp::Lt => Value::Bool(x < y),
                BinOp::Gt => Value::Bool(x > y),
                BinOp::Le => Value::Bool(x <= y),
                BinOp::Ge => Value::Bool(x >= y),
                BinOp::Eq => Value::Bool(x == y),
                BinOp::Ne => Value::Bool(x != y),
                _ => return self.binop_slow(op, a, b),
            });
        }
        self.binop_slow(op, a, b)
    }

    fn binop_slow(&mut self, op: BinOp, a: Value, b: Value) -> JsResult<Value> {
        Ok(match op {
            BinOp::Add => {
                if let (Value::Str(x), Value::Str(y)) = (&a, &b) {
                    return Ok(Value::Str(concat(x, y)));
                }
                let pa = self.to_primitive(a, None)?;
                let pb = self.to_primitive(b, None)?;
                if matches!(pa, Value::Str(_)) || matches!(pb, Value::Str(_)) {
                    let sa = self.to_string(pa)?;
                    let sb = self.to_string(pb)?;
                    Value::Str(concat(&sa, &sb))
                } else {
                    Value::Num(self.to_number(pa)? + self.to_number(pb)?)
                }
            }
            BinOp::Sub | BinOp::Mul | BinOp::Div | BinOp::Mod => {
                let x = self.to_number(a)?;
                let y = self.to_number(b)?;
                Value::Num(match op {
                    BinOp::Sub => x - y,
                    BinOp::Mul => x * y,
                    BinOp::Div => x / y,
                    _ => x % y,
                })
            }
            BinOp::Shl | BinOp::Sar | BinOp::Shr => {
                let x = self.to_number(a)?;
                let y = self.to_number(b)?;
                let s = to_uint32(y) & 31;
                Value::Num(match op {
                    BinOp::Shl => to_int32(x).wrapping_shl(s) as f64,
                    BinOp::Sar => (to_int32(x) >> s) as f64,
                    _ => (to_uint32(x) >> s) as f64,
                })
            }
            BinOp::BitAnd | BinOp::BitXor | BinOp::BitOr => {
                let x = to_int32(self.to_number(a)?);
                let y = to_int32(self.to_number(b)?);
                Value::Num(match op {
                    BinOp::BitAnd => x & y,
                    BinOp::BitXor => x ^ y,
                    _ => x | y,
                } as f64)
            }
            BinOp::Lt => Value::Bool(self.compare(a, b)? == Some(true)),
            BinOp::Gt => Value::Bool(self.compare(b, a)? == Some(true)),
            BinOp::Le => Value::Bool(self.compare(b, a)? == Some(false)),
            BinOp::Ge => Value::Bool(self.compare(a, b)? == Some(false)),
            BinOp::Eq => Value::Bool(self.equals(a, b)?),
            BinOp::Ne => Value::Bool(!self.equals(a, b)?),
        })
    }

    /// The abstract relational comparison x < y (§11.8.5); x is always converted first, so for
    /// > and <= (which compare right < left) the right operand is converted first.
    fn compare(&mut self, x: Value, y: Value) -> JsResult<Option<bool>> {
        let px = self.to_primitive(x, Some(Hint::Number))?;
        let py = self.to_primitive(y, Some(Hint::Number))?;
        if let (Value::Str(a), Value::Str(b)) = (&px, &py) {
            return Ok(Some(a[..] < b[..]));
        }
        let nx = self.to_number(px)?;
        let ny = self.to_number(py)?;
        if nx.is_nan() || ny.is_nan() {
            return Ok(None);
        }
        Ok(Some(nx < ny))
    }

    /// The abstract equality comparison (§11.9.3).
    pub fn equals(&mut self, x: Value, y: Value) -> JsResult<bool> {
        use Value::*;
        Ok(match (&x, &y) {
            (Undef, Undef) | (Null, Null) | (Undef, Null) | (Null, Undef) => true,
            (Num(a), Num(b)) => a == b,
            (Str(a), Str(b)) => a == b,
            (Bool(a), Bool(b)) => a == b,
            (Obj(a), Obj(b)) => Rc::ptr_eq(a, b),
            (Num(a), Str(b)) => *a == str_to_number(b),
            (Str(a), Num(b)) => str_to_number(a) == *b,
            (Bool(a), _) => {
                let n = *a as u8 as f64;
                return self.equals(Num(n), y);
            }
            (_, Bool(b)) => {
                let n = *b as u8 as f64;
                return self.equals(x, Num(n));
            }
            (Str(_) | Num(_), Obj(_)) => {
                let p = self.to_primitive(y, None)?;
                return self.equals(x, p);
            }
            (Obj(_), Str(_) | Num(_)) => {
                let p = self.to_primitive(x, None)?;
                return self.equals(p, y);
            }
            _ => false,
        })
    }
}
