// Native ECMAScript objects (chapter 15) and the host's print function.

use std::rc::Rc;

use crate::date;
use crate::interp::*;
use crate::numconv::*;
use crate::parser::{make_decl, parse_program, Parser};
use crate::value::*;

use Value::*;

fn mkfn(it: &mut Interp, name: &'static str, len: u32, call: NativeFn, ctor: Option<NativeFn>) -> ObjRef {
    let o = Object::new("Function", Some(it.function_proto.clone()), Kind::Func(Func::Native(Native { name, call, ctor })));
    o.borrow_mut().set(it.n.length.clone(), Num(len as f64), RO | DD | DE);
    o
}

fn method(it: &mut Interp, o: &ObjRef, name: &'static str, len: u32, f: NativeFn) -> ObjRef {
    let fo = mkfn(it, name, len, f, None);
    o.borrow_mut().set(intern_str(name), Obj(fo.clone()), DE);
    fo
}

fn constant(o: &ObjRef, name: &str, v: Value) {
    o.borrow_mut().set(intern_str(name), v, RO | DD | DE);
}

fn link(it: &mut Interp, ctor: &ObjRef, proto: &ObjRef) {
    ctor.borrow_mut().set(it.n.prototype.clone(), Obj(proto.clone()), RO | DD | DE);
    proto.borrow_mut().set(it.n.constructor.clone(), Obj(ctor.clone()), DE);
}

fn num_arg(it: &mut Interp, args: &[Value], i: usize) -> JsResult<f64> {
    it.to_number(arg(args, i))
}

fn str_val(s: &str) -> Value {
    Str(js(s))
}

pub fn setup(it: &mut Interp) {
    let g = it.global.clone();
    {
        let mut gb = g.borrow_mut();
        gb.set(intern_str("NaN"), Num(f64::NAN), DE);
        gb.set(intern_str("Infinity"), Num(f64::INFINITY), DE);
        // Not in ES1 §15.1.1; §2 permits additional properties of built-in objects.
        gb.set(intern_str("undefined"), Undef, DE);
    }
    method(it, &g, "eval", 1, global_eval);
    method(it, &g, "parseInt", 2, parse_int);
    method(it, &g, "parseFloat", 1, parse_float);
    method(it, &g, "escape", 1, escape);
    method(it, &g, "unescape", 1, unescape);
    method(it, &g, "isNaN", 1, |it, _, a| Ok(Bool(num_arg(it, a, 0)?.is_nan())));
    method(it, &g, "isFinite", 1, |it, _, a| Ok(Bool(num_arg(it, a, 0)?.is_finite())));
    method(it, &g, "print", 0, print);

    // Object (§15.2).
    let op = it.object_proto.clone();
    let object = mkfn(it, "Object", 1, object_call, Some(object_construct));
    link(it, &object, &op);
    method(it, &op, "toString", 0, object_to_string);
    method(it, &op, "valueOf", 0, |it, this, _| Ok(Obj(it.to_object(this)?)));
    g.borrow_mut().set(intern_str("Object"), Obj(object), DE);

    // Function (§15.3).
    let fp = it.function_proto.clone();
    fp.borrow_mut().set(it.n.length.clone(), Num(0.0), RO | DD | DE);
    let function = mkfn(it, "Function", 1, function_construct, Some(function_construct));
    link(it, &function, &fp);
    method(it, &fp, "toString", 0, function_to_string);
    g.borrow_mut().set(intern_str("Function"), Obj(function), DE);

    // Array (§15.4).
    let ap = it.array_proto.clone();
    let array = mkfn(it, "Array", 1, array_construct, Some(array_construct));
    link(it, &array, &ap);
    method(it, &ap, "toString", 0, |it, this, _| array_join(it, this, &[]));
    method(it, &ap, "join", 1, array_join);
    method(it, &ap, "reverse", 0, array_reverse);
    method(it, &ap, "sort", 1, array_sort);
    g.borrow_mut().set(intern_str("Array"), Obj(array), DE);

    // String (§15.5).
    let sp = it.string_proto.clone();
    let string = mkfn(it, "String", 1, string_call, Some(string_construct));
    link(it, &string, &sp);
    method(it, &string, "fromCharCode", 1, string_from_char_code);
    method(it, &sp, "toString", 0, string_value_of);
    method(it, &sp, "valueOf", 0, string_value_of);
    method(it, &sp, "charAt", 1, string_char_at);
    method(it, &sp, "charCodeAt", 1, string_char_code_at);
    method(it, &sp, "indexOf", 2, string_index_of);
    method(it, &sp, "lastIndexOf", 2, string_last_index_of);
    method(it, &sp, "split", 1, string_split);
    method(it, &sp, "substring", 2, string_substring);
    method(it, &sp, "toLowerCase", 0, |it, this, _| string_case(it, this, false));
    method(it, &sp, "toUpperCase", 0, |it, this, _| string_case(it, this, true));
    g.borrow_mut().set(intern_str("String"), Obj(string), DE);

    // Boolean (§15.6).
    let bp = it.boolean_proto.clone();
    let boolean = mkfn(it, "Boolean", 1, |_, _, a| Ok(Bool(to_boolean(&arg(a, 0)))), Some(boolean_construct));
    link(it, &boolean, &bp);
    method(it, &bp, "toString", 0, |it, this, _| {
        let b = this_bool(&this)?;
        Ok(Str(if b { it.n.true_.clone() } else { it.n.false_.clone() }))
    });
    method(it, &bp, "valueOf", 0, |_, this, _| Ok(Bool(this_bool(&this)?)));
    g.borrow_mut().set(intern_str("Boolean"), Obj(boolean), DE);

    // Number (§15.7).
    let np = it.number_proto.clone();
    let number = mkfn(it, "Number", 1, number_call, Some(number_construct));
    link(it, &number, &np);
    constant(&number, "MAX_VALUE", Num(f64::MAX));
    constant(&number, "MIN_VALUE", Num(5e-324));
    constant(&number, "NaN", Num(f64::NAN));
    constant(&number, "NEGATIVE_INFINITY", Num(f64::NEG_INFINITY));
    constant(&number, "POSITIVE_INFINITY", Num(f64::INFINITY));
    method(it, &np, "toString", 1, number_to_string);
    method(it, &np, "valueOf", 0, |_, this, _| Ok(Num(this_num(&this)?)));
    g.borrow_mut().set(intern_str("Number"), Obj(number), DE);

    // Math (§15.8).
    let math = it.new_object();
    constant(&math, "E", Num(std::f64::consts::E));
    constant(&math, "LN10", Num(std::f64::consts::LN_10));
    constant(&math, "LN2", Num(std::f64::consts::LN_2));
    constant(&math, "LOG2E", Num(std::f64::consts::LOG2_E));
    constant(&math, "LOG10E", Num(std::f64::consts::LOG10_E));
    constant(&math, "PI", Num(std::f64::consts::PI));
    constant(&math, "SQRT1_2", Num(std::f64::consts::FRAC_1_SQRT_2));
    constant(&math, "SQRT2", Num(std::f64::consts::SQRT_2));
    method(it, &math, "abs", 1, |it, _, a| Ok(Num(num_arg(it, a, 0)?.abs())));
    method(it, &math, "acos", 1, |it, _, a| Ok(Num(num_arg(it, a, 0)?.acos())));
    method(it, &math, "asin", 1, |it, _, a| Ok(Num(num_arg(it, a, 0)?.asin())));
    method(it, &math, "atan", 1, |it, _, a| Ok(Num(num_arg(it, a, 0)?.atan())));
    method(it, &math, "atan2", 2, |it, _, a| {
        let y = num_arg(it, a, 0)?;
        let x = num_arg(it, a, 1)?;
        Ok(Num(y.atan2(x)))
    });
    method(it, &math, "ceil", 1, |it, _, a| Ok(Num(num_arg(it, a, 0)?.ceil())));
    method(it, &math, "cos", 1, |it, _, a| Ok(Num(num_arg(it, a, 0)?.cos())));
    method(it, &math, "exp", 1, |it, _, a| Ok(Num(num_arg(it, a, 0)?.exp())));
    method(it, &math, "floor", 1, |it, _, a| Ok(Num(num_arg(it, a, 0)?.floor())));
    method(it, &math, "log", 1, |it, _, a| Ok(Num(num_arg(it, a, 0)?.ln())));
    method(it, &math, "max", 2, |it, _, a| {
        let x = num_arg(it, a, 0)?;
        let y = num_arg(it, a, 1)?;
        Ok(Num(if x.is_nan() || y.is_nan() {
            f64::NAN
        } else if x == 0.0 && y == 0.0 {
            if x.is_sign_negative() && y.is_sign_negative() { -0.0 } else { 0.0 }
        } else if x > y {
            x
        } else {
            y
        }))
    });
    method(it, &math, "min", 2, |it, _, a| {
        let x = num_arg(it, a, 0)?;
        let y = num_arg(it, a, 1)?;
        Ok(Num(if x.is_nan() || y.is_nan() {
            f64::NAN
        } else if x == 0.0 && y == 0.0 {
            if x.is_sign_negative() || y.is_sign_negative() { -0.0 } else { 0.0 }
        } else if x < y {
            x
        } else {
            y
        }))
    });
    method(it, &math, "pow", 2, |it, _, a| {
        let x = num_arg(it, a, 0)?;
        let y = num_arg(it, a, 1)?;
        Ok(Num(if y.is_nan() {
            f64::NAN
        } else if y == 0.0 {
            1.0
        } else if x.is_nan() || (x.abs() == 1.0 && y.is_infinite()) {
            f64::NAN
        } else {
            x.powf(y)
        }))
    });
    method(it, &math, "random", 0, |it, _, _| {
        let mut x = it.rng;
        x ^= x >> 12;
        x ^= x << 25;
        x ^= x >> 27;
        it.rng = x;
        let r = x.wrapping_mul(0x2545F4914F6CDD1D) >> 11;
        Ok(Num(r as f64 / (1u64 << 53) as f64))
    });
    method(it, &math, "round", 1, |it, _, a| {
        let x = num_arg(it, a, 0)?;
        Ok(Num(if !x.is_finite() || x == 0.0 {
            x
        } else if x > 0.0 && x < 0.5 {
            0.0
        } else if x < 0.0 && x >= -0.5 {
            -0.0
        } else {
            let f = x.floor();
            if x - f >= 0.5 { f + 1.0 } else { f }
        }))
    });
    method(it, &math, "sin", 1, |it, _, a| Ok(Num(num_arg(it, a, 0)?.sin())));
    method(it, &math, "sqrt", 1, |it, _, a| Ok(Num(num_arg(it, a, 0)?.sqrt())));
    method(it, &math, "tan", 1, |it, _, a| Ok(Num(num_arg(it, a, 0)?.tan())));
    g.borrow_mut().set(intern_str("Math"), Obj(math), DE);

    // Date (§15.9).
    let dp = it.date_proto.clone();
    let datec = mkfn(it, "Date", 7, date_call, Some(date_construct));
    link(it, &datec, &dp);
    method(it, &datec, "parse", 1, |it, _, a| {
        let s = it.to_string(arg(a, 0))?;
        Ok(Num(date::parse(&rs(&s))))
    });
    method(it, &datec, "UTC", 7, |it, _, a| {
        let t = date_from_components(it, a)?;
        Ok(Num(date::time_clip(t)))
    });
    setup_date_proto(it, &dp);
    g.borrow_mut().set(intern_str("Date"), Obj(datec), DE);
}

// Global functions (§15.1.2).

fn global_eval(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    let s = match arg(args, 0) {
        Str(s) => s,
        other => return Ok(other),
    };
    let prog = match parse_program(&rs(&s)) {
        Ok(p) => p,
        Err(e) => return throw(format!("SyntaxError: {} (in eval, line {})", e.msg, e.line)),
    };
    Ok(it.run_program(&prog)?.unwrap_or(Undef))
}

fn skip_ws(s: &[u16]) -> &[u16] {
    let mut i = 0;
    while i < s.len() && is_str_ws(s[i]) {
        i += 1;
    }
    &s[i..]
}

fn parse_int(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    let st = it.to_string(arg(args, 0))?;
    let mut s = skip_ws(&st);
    let mut sign = 1.0;
    if let Some(&c) = s.first() {
        if c == b'-' as u16 {
            sign = -1.0;
        }
        if c == b'-' as u16 || c == b'+' as u16 {
            s = &s[1..];
        }
    }
    let mut r = to_int32(num_arg(it, args, 1)?);
    let has_0x = |s: &[u16]| s.len() >= 2 && s[0] == b'0' as u16 && (s[1] == b'x' as u16 || s[1] == b'X' as u16);
    if r == 0 {
        if has_0x(s) {
            s = &s[2..];
            r = 16;
        } else if s.first() == Some(&(b'0' as u16)) {
            r = 8;
        } else {
            r = 10;
        }
    } else if !(2..=36).contains(&r) {
        return Ok(Num(f64::NAN));
    } else if r == 16 && has_0x(s) {
        s = &s[2..];
    }
    let digit = |c: u16| -> Option<u32> { char::from_u32(c as u32).and_then(|ch| ch.to_digit(36)).filter(|d| (*d as i32) < r) };
    let n = s.iter().take_while(|c| digit(**c).is_some()).count();
    if n == 0 {
        return Ok(Num(f64::NAN));
    }
    let z = &s[..n];
    let v = if r == 10 {
        parse_decimal(z)
    } else {
        z.iter().fold(0.0, |acc, c| acc * r as f64 + digit(*c).unwrap() as f64)
    };
    Ok(Num(sign * v))
}

fn parse_float(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    let st = it.to_string(arg(args, 0))?;
    let s = skip_ws(&st);
    let n = decimal_prefix(s);
    if n == 0 {
        return Ok(Num(f64::NAN));
    }
    Ok(Num(parse_decimal(&s[..n])))
}

fn escape(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    let s = it.to_string(arg(args, 0))?;
    let mut out = String::new();
    for &c in s.iter() {
        let ch = char::from_u32(c as u32).unwrap_or('\0');
        if c < 128 && (ch.is_ascii_alphanumeric() || "@*_+-./".contains(ch)) {
            out.push(ch);
        } else if c < 256 {
            out.push_str(&format!("%{:02X}", c));
        } else {
            out.push_str(&format!("%u{:04X}", c));
        }
    }
    Ok(str_val(&out))
}

fn unescape(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    let s = it.to_string(arg(args, 0))?;
    let hex = |i: usize| s.get(i).and_then(|c| crate::lexer::hex_val(*c as u32));
    let mut out: Vec<u16> = Vec::with_capacity(s.len());
    let mut k = 0;
    while k < s.len() {
        let c = s[k];
        if c == b'%' as u16 {
            if s.get(k + 1) == Some(&(b'u' as u16)) {
                if let (Some(a), Some(b), Some(c2), Some(d)) = (hex(k + 2), hex(k + 3), hex(k + 4), hex(k + 5)) {
                    out.push((a << 12 | b << 8 | c2 << 4 | d) as u16);
                    k += 6;
                    continue;
                }
            }
            if let (Some(a), Some(b)) = (hex(k + 1), hex(k + 2)) {
                out.push((a << 4 | b) as u16);
                k += 3;
                continue;
            }
        }
        out.push(c);
        k += 1;
    }
    Ok(Str(out.into()))
}

fn print(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    let mut line = String::new();
    for (i, a) in args.iter().enumerate() {
        if i > 0 {
            line.push(' ');
        }
        let s = it.to_string(a.clone())?;
        line.push_str(&rs(&s));
    }
    line.push('\n');
    it.write_out(&line);
    Ok(Undef)
}

// Object (§15.2).

fn object_call(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    match arg(args, 0) {
        Undef | Null => Ok(Obj(it.new_object())),
        v => Ok(Obj(it.to_object(v)?)),
    }
}

fn object_construct(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    match arg(args, 0) {
        Undef | Null => Ok(Obj(it.new_object())),
        v => Ok(Obj(it.to_object(v)?)),
    }
}

fn class_of(v: &Value) -> &'static str {
    match v {
        Obj(o) => o.borrow().class,
        Str(_) => "String",
        Num(_) => "Number",
        Bool(_) => "Boolean",
        _ => "Object",
    }
}

fn object_to_string(_: &mut Interp, this: Value, _: &[Value]) -> JsResult<Value> {
    Ok(str_val(&format!("[object {}]", class_of(&this))))
}

// Function (§15.3).

fn function_construct(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    let mut p = String::new();
    let mut body = String::new();
    if let Some((last, params)) = args.split_last() {
        for (i, a) in params.iter().enumerate() {
            if i > 0 {
                p.push(',');
            }
            let s = it.to_string(a.clone())?;
            p.push_str(&rs(&s));
        }
        body = rs(&it.to_string(last.clone())?);
    }
    let pchars: Vec<char> = p.chars().collect();
    let params = match Parser::new(&pchars).and_then(|ps| ps.parse_params()) {
        Ok(v) => v,
        Err(e) => return throw(format!("SyntaxError: bad parameter list for Function: {}", e.msg)),
    };
    let bchars: Vec<char> = body.chars().collect();
    let (stmts, vars) = match Parser::new(&bchars).and_then(|ps| ps.parse_body()) {
        Ok(v) => v,
        Err(e) => return throw(format!("SyntaxError: bad body for Function: {}", e.msg)),
    };
    let source = format!("function anonymous({}\n) {{\n{}\n}}", p, body);
    let decl = Rc::new(make_decl(intern_str("anonymous"), params, stmts, vars, source));
    Ok(Obj(it.make_function(decl)))
}

fn function_to_string(_: &mut Interp, this: Value, _: &[Value]) -> JsResult<Value> {
    if let Obj(o) = &this {
        if let Kind::Func(f) = &o.borrow().kind {
            let s = match f {
                Func::User(d) => d.source.clone(),
                Func::Native(n) => {
                    let name = if n.name.is_empty() { "anonymous" } else { n.name };
                    format!("function {}() {{\n    /* native code */\n}}", name)
                }
            };
            return Ok(str_val(&s));
        }
    }
    throw("TypeError: Function.prototype.toString called on an object that is not a function")
}

// Array (§15.4).

fn array_construct(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    let a = it.new_array();
    let len_key = it.n.length.clone();
    if args.len() == 1 {
        match &args[0] {
            Num(n) => a.borrow_mut().set(len_key, Num(to_uint32(*n) as f64), DE | DD),
            v => {
                let k = it.index_key(0);
                let mut b = a.borrow_mut();
                b.set(k, v.clone(), 0);
                b.set(len_key, Num(1.0), DE | DD);
            }
        }
    } else {
        for (i, v) in args.iter().enumerate() {
            let k = it.index_key(i);
            a.borrow_mut().set(k, v.clone(), 0);
        }
        a.borrow_mut().set(len_key, Num(args.len() as f64), DE | DD);
    }
    Ok(Obj(a))
}

fn length_of(it: &mut Interp, o: &ObjRef) -> JsResult<u32> {
    let l = get(o, &it.n.length);
    Ok(to_uint32(it.to_number(l)?))
}

fn array_join(it: &mut Interp, this: Value, args: &[Value]) -> JsResult<Value> {
    let o = it.to_object(this)?;
    let len = length_of(it, &o)?;
    let sep: JsStr = if args.is_empty() { js(",") } else { it.to_string(args[0].clone())? };
    let mut out: Vec<u16> = Vec::new();
    for k in 0..len {
        if k > 0 {
            out.extend_from_slice(&sep);
        }
        let key = it.index_key(k as usize);
        let e = get(&o, &key);
        if !matches!(e, Undef | Null) {
            let s = it.to_string(e)?;
            out.extend_from_slice(&s);
        }
    }
    Ok(Str(out.into()))
}

fn array_reverse(it: &mut Interp, this: Value, _: &[Value]) -> JsResult<Value> {
    let o = it.to_object(this)?;
    let len = length_of(it, &o)? as usize;
    let mid = len / 2;
    for lower in 0..mid {
        let upper = len - lower - 1;
        let lk = it.index_key(lower);
        let uk = it.index_key(upper);
        let lv = get(&o, &lk);
        let uv = get(&o, &uk);
        let lh = has_property(&o, &lk);
        let uh = has_property(&o, &uk);
        match (lh, uh) {
            (true, true) => {
                it.put(&o, &lk, uv)?;
                it.put(&o, &uk, lv)?;
            }
            (false, true) => {
                it.put(&o, &lk, uv)?;
                delete(&o, &uk);
            }
            (true, false) => {
                delete(&o, &lk);
                it.put(&o, &uk, lv)?;
            }
            (false, false) => {}
        }
    }
    Ok(Obj(o))
}

fn sort_compare(it: &mut Interp, x: &Value, y: &Value, cmp: &Value) -> JsResult<f64> {
    if let Undef = cmp {
        let a = it.to_string(x.clone())?;
        let b = it.to_string(y.clone())?;
        return Ok(match a[..].cmp(&b[..]) {
            std::cmp::Ordering::Less => -1.0,
            std::cmp::Ordering::Greater => 1.0,
            std::cmp::Ordering::Equal => 0.0,
        });
    }
    let r = it.call(cmp, Undef, &[x.clone(), y.clone()])?;
    let n = it.to_number(r)?;
    Ok(if n.is_nan() { 0.0 } else { n })
}

fn merge_sort(it: &mut Interp, mut v: Vec<Value>, cmp: &Value) -> JsResult<Vec<Value>> {
    if v.len() <= 1 {
        return Ok(v);
    }
    let right = v.split_off(v.len() / 2);
    let left = merge_sort(it, v, cmp)?;
    let right = merge_sort(it, right, cmp)?;
    let mut out = Vec::with_capacity(left.len() + right.len());
    let (mut i, mut j) = (0, 0);
    while i < left.len() && j < right.len() {
        if sort_compare(it, &right[j], &left[i], cmp)? < 0.0 {
            out.push(right[j].clone());
            j += 1;
        } else {
            out.push(left[i].clone());
            i += 1;
        }
    }
    out.extend_from_slice(&left[i..]);
    out.extend_from_slice(&right[j..]);
    Ok(out)
}

fn array_sort(it: &mut Interp, this: Value, args: &[Value]) -> JsResult<Value> {
    let o = it.to_object(this)?;
    let len = length_of(it, &o)? as usize;
    let cmp = arg(args, 0);
    let mut defined = Vec::new();
    let mut undefs = 0;
    for k in 0..len {
        let key = it.index_key(k);
        if has_property(&o, &key) {
            match get(&o, &key) {
                Undef => undefs += 1,
                v => defined.push(v),
            }
        }
    }
    let sorted = merge_sort(it, defined, &cmp)?;
    let n = sorted.len();
    for (k, v) in sorted.into_iter().enumerate() {
        let key = it.index_key(k);
        it.put(&o, &key, v)?;
    }
    for k in n..n + undefs {
        let key = it.index_key(k);
        it.put(&o, &key, Undef)?;
    }
    for k in n + undefs..len {
        let key = it.index_key(k);
        delete(&o, &key);
    }
    Ok(Obj(o))
}

// String (§15.5).

fn string_call(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    if args.is_empty() {
        return Ok(Str(it.n.empty.clone()));
    }
    Ok(Str(it.to_string(args[0].clone())?))
}

fn string_construct(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    let s = if args.is_empty() { it.n.empty.clone() } else { it.to_string(args[0].clone())? };
    Ok(Obj(it.new_string_object(s)))
}

fn string_from_char_code(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    let mut v = Vec::with_capacity(args.len());
    for a in args {
        v.push(to_uint16(it.to_number(a.clone())?));
    }
    Ok(Str(v.into()))
}

fn string_value_of(_: &mut Interp, this: Value, _: &[Value]) -> JsResult<Value> {
    match &this {
        Str(s) => Ok(Str(s.clone())),
        Obj(o) => match &o.borrow().kind {
            Kind::Str(s) => Ok(Str(s.clone())),
            _ => throw("TypeError: String.prototype method called on an object that is not a String"),
        },
        _ => throw("TypeError: String.prototype method called on a value that is not a String"),
    }
}

fn string_char_at(it: &mut Interp, this: Value, args: &[Value]) -> JsResult<Value> {
    let s = it.to_string(this)?;
    let p = to_integer(num_arg(it, args, 0)?);
    if p < 0.0 || p >= s.len() as f64 {
        return Ok(Str(it.n.empty.clone()));
    }
    Ok(Str(s[p as usize..p as usize + 1].into()))
}

fn string_char_code_at(it: &mut Interp, this: Value, args: &[Value]) -> JsResult<Value> {
    let s = it.to_string(this)?;
    let p = to_integer(num_arg(it, args, 0)?);
    if p < 0.0 || p >= s.len() as f64 {
        return Ok(Num(f64::NAN));
    }
    Ok(Num(s[p as usize] as f64))
}

fn find_at(s: &[u16], t: &[u16], k: usize) -> bool {
    k + t.len() <= s.len() && &s[k..k + t.len()] == t
}

fn string_index_of(it: &mut Interp, this: Value, args: &[Value]) -> JsResult<Value> {
    let s = it.to_string(this)?;
    let t = it.to_string(arg(args, 0))?;
    let pos = to_integer(num_arg(it, args, 1)?);
    let start = pos.max(0.0).min(s.len() as f64) as usize;
    for k in start..=s.len() {
        if find_at(&s, &t, k) {
            return Ok(Num(k as f64));
        }
    }
    Ok(Num(-1.0))
}

fn string_last_index_of(it: &mut Interp, this: Value, args: &[Value]) -> JsResult<Value> {
    let s = it.to_string(this)?;
    let t = it.to_string(arg(args, 0))?;
    let n = num_arg(it, args, 1)?;
    let pos = if n.is_nan() { f64::INFINITY } else { to_integer(n) };
    let start = pos.max(0.0).min(s.len() as f64) as usize;
    for k in (0..=start).rev() {
        if find_at(&s, &t, k) {
            return Ok(Num(k as f64));
        }
    }
    Ok(Num(-1.0))
}

fn string_split(it: &mut Interp, this: Value, args: &[Value]) -> JsResult<Value> {
    let s = it.to_string(this)?;
    let a = it.new_array();
    let mut parts: Vec<JsStr> = Vec::new();
    if args.is_empty() {
        parts.push(s);
    } else {
        let sep = it.to_string(args[0].clone())?;
        if sep.is_empty() {
            for k in 0..s.len() {
                parts.push(s[k..k + 1].into());
            }
        } else {
            let mut p = 0;
            let mut k = 0;
            while k + sep.len() <= s.len() {
                if find_at(&s, &sep, k) {
                    parts.push(s[p..k].into());
                    k += sep.len();
                    p = k;
                } else {
                    k += 1;
                }
            }
            parts.push(s[p..].into());
        }
    }
    for (i, part) in parts.into_iter().enumerate() {
        let k = it.index_key(i);
        it.put(&a, &k, Str(part))?;
    }
    Ok(Obj(a))
}

fn string_substring(it: &mut Interp, this: Value, args: &[Value]) -> JsResult<Value> {
    let s = it.to_string(this)?;
    let len = s.len() as f64;
    let start = to_integer(num_arg(it, args, 0)?).max(0.0).min(len);
    let end = if args.len() < 2 { len } else { to_integer(num_arg(it, args, 1)?).max(0.0).min(len) };
    let (a, b) = if start <= end { (start, end) } else { (end, start) };
    Ok(Str(s[a as usize..b as usize].into()))
}

fn string_case(it: &mut Interp, this: Value, upper: bool) -> JsResult<Value> {
    let s = it.to_string(this)?;
    let v: Vec<u16> = s
        .iter()
        .map(|&c| {
            let ch = match char::from_u32(c as u32) {
                Some(ch) => ch,
                None => return c,
            };
            let mut m: Vec<char> = if upper { ch.to_uppercase().collect() } else { ch.to_lowercase().collect() };
            if m.len() == 1 && (m[0] as u32) < 0x10000 { m.pop().unwrap() as u32 as u16 } else { c }
        })
        .collect();
    Ok(Str(v.into()))
}

// Boolean (§15.6) and Number (§15.7).

fn boolean_construct(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    let b = to_boolean(&arg(args, 0));
    Ok(Obj(Object::new("Boolean", Some(it.boolean_proto.clone()), Kind::Bool(b))))
}

fn this_bool(this: &Value) -> JsResult<bool> {
    match this {
        Bool(b) => Ok(*b),
        Obj(o) => match o.borrow().kind {
            Kind::Bool(b) => Ok(b),
            _ => throw("TypeError: Boolean.prototype method called on an object that is not a Boolean"),
        },
        _ => throw("TypeError: Boolean.prototype method called on a value that is not a Boolean"),
    }
}

fn number_call(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    if args.is_empty() {
        return Ok(Num(0.0));
    }
    Ok(Num(it.to_number(args[0].clone())?))
}

fn number_construct(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    let n = if args.is_empty() { 0.0 } else { it.to_number(args[0].clone())? };
    Ok(Obj(Object::new("Number", Some(it.number_proto.clone()), Kind::Num(n))))
}

fn this_num(this: &Value) -> JsResult<f64> {
    match this {
        Num(n) => Ok(*n),
        Obj(o) => match o.borrow().kind {
            Kind::Num(n) => Ok(n),
            _ => throw("TypeError: Number.prototype method called on an object that is not a Number"),
        },
        _ => throw("TypeError: Number.prototype method called on a value that is not a Number"),
    }
}

fn number_to_string(it: &mut Interp, this: Value, args: &[Value]) -> JsResult<Value> {
    let n = this_num(&this)?;
    let radix = match arg(args, 0) {
        Undef => 10.0,
        r => to_integer(it.to_number(r)?),
    };
    if radix == 10.0 || !(2.0..=36.0).contains(&radix) {
        return Ok(str_val(&num_to_string(n)));
    }
    Ok(str_val(&num_to_radix(n, radix as u32)))
}

// Date (§15.9).

fn now() -> f64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_millis() as f64)
        .unwrap_or_else(|e| -(e.duration().as_millis() as f64))
}

fn date_call(_: &mut Interp, _: Value, _: &[Value]) -> JsResult<Value> {
    Ok(str_val(&date::to_string(date::time_clip(now()))))
}

/// MakeDate(MakeDay(...), MakeTime(...)) from (year, month[, date[, hours[, minutes[, seconds[, ms]]]]]).
fn date_from_components(it: &mut Interp, args: &[Value]) -> JsResult<f64> {
    let mut c = [f64::NAN, f64::NAN, 1.0, 0.0, 0.0, 0.0, 0.0];
    let n = args.len().max(2).min(7);
    for (i, slot) in c.iter_mut().enumerate().take(n) {
        *slot = num_arg(it, args, i)?;
    }
    let mut y = c[0];
    if !y.is_nan() {
        let yi = to_integer(y);
        if (0.0..=99.0).contains(&yi) {
            y = 1900.0 + yi;
        }
    }
    Ok(date::make_date(date::make_day(y, c[1], c[2]), date::make_time(c[3], c[4], c[5], c[6])))
}

fn new_date(it: &mut Interp, t: f64) -> Value {
    Obj(Object::new("Date", Some(it.date_proto.clone()), Kind::Date(t)))
}

fn date_construct(it: &mut Interp, _: Value, args: &[Value]) -> JsResult<Value> {
    let t = match args.len() {
        0 => date::time_clip(now()),
        1 => {
            let p = it.to_primitive(args[0].clone(), None)?;
            match p {
                Str(s) => date::parse(&rs(&s)),
                v => date::time_clip(it.to_number(v)?),
            }
        }
        _ => {
            let t = date_from_components(it, args)?;
            date::time_clip(date::utc(t))
        }
    };
    Ok(new_date(it, t))
}

fn this_time(this: &Value) -> JsResult<(ObjRef, f64)> {
    if let Obj(o) = this {
        if let Kind::Date(t) = o.borrow().kind {
            return Ok((o.clone(), t));
        }
    }
    throw("TypeError: Date.prototype method called on an object that is not a Date")
}

fn set_time(o: &ObjRef, t: f64) -> Value {
    o.borrow_mut().kind = Kind::Date(t);
    Num(t)
}

/// The setters: replaces `count` components starting at `first`
/// (0 year, 1 month, 2 date, 3 hours, 4 minutes, 5 seconds, 6 ms).
fn date_setter(it: &mut Interp, this: Value, args: &[Value], local: bool, first: usize, count: usize) -> JsResult<Value> {
    let (o, t) = this_time(&this)?;
    let mut vals = Vec::new();
    for i in 0..args.len().min(count).max(1) {
        vals.push(num_arg(it, args, i)?);
    }
    let mut t = if local { date::local_time(t) } else { t };
    if first == 0 && t.is_nan() {
        t = 0.0;
    }
    let mut c = [
        date::year_from_time(t),
        date::month_from_time(t),
        date::date_from_time(t),
        date::hour_from_time(t),
        date::min_from_time(t),
        date::sec_from_time(t),
        date::ms_from_time(t),
    ];
    if t.is_nan() {
        c = [f64::NAN; 7];
    }
    for (i, v) in vals.into_iter().enumerate() {
        c[first + i] = v;
    }
    let r = date::make_date(date::make_day(c[0], c[1], c[2]), date::make_time(c[3], c[4], c[5], c[6]));
    let r = if local { date::utc(r) } else { r };
    Ok(set_time(&o, date::time_clip(r)))
}

fn date_getter(this: &Value, local: bool, f: fn(f64) -> f64) -> JsResult<Value> {
    let (_, t) = this_time(this)?;
    if t.is_nan() {
        return Ok(Num(f64::NAN));
    }
    let t = if local { date::local_time(t) } else { t };
    Ok(Num(f(t)))
}

fn setup_date_proto(it: &mut Interp, dp: &ObjRef) {
    method(it, dp, "toString", 0, |_, this, _| Ok(str_val(&date::to_string(this_time(&this)?.1))));
    method(it, dp, "toLocaleString", 0, |_, this, _| Ok(str_val(&date::to_string(this_time(&this)?.1))));
    let utc = method(it, dp, "toUTCString", 0, |_, this, _| Ok(str_val(&date::to_utc_string(this_time(&this)?.1))));
    dp.borrow_mut().set(intern_str("toGMTString"), Obj(utc), DE);
    method(it, dp, "valueOf", 0, |_, this, _| Ok(Num(this_time(&this)?.1)));
    method(it, dp, "getTime", 0, |_, this, _| Ok(Num(this_time(&this)?.1)));
    method(it, dp, "getYear", 0, |_, this, _| date_getter(&this, true, |t| date::year_from_time(t) - 1900.0));
    method(it, dp, "getFullYear", 0, |_, this, _| date_getter(&this, true, date::year_from_time));
    method(it, dp, "getUTCFullYear", 0, |_, this, _| date_getter(&this, false, date::year_from_time));
    method(it, dp, "getMonth", 0, |_, this, _| date_getter(&this, true, date::month_from_time));
    method(it, dp, "getUTCMonth", 0, |_, this, _| date_getter(&this, false, date::month_from_time));
    method(it, dp, "getDate", 0, |_, this, _| date_getter(&this, true, date::date_from_time));
    method(it, dp, "getUTCDate", 0, |_, this, _| date_getter(&this, false, date::date_from_time));
    method(it, dp, "getDay", 0, |_, this, _| date_getter(&this, true, date::week_day));
    method(it, dp, "getUTCDay", 0, |_, this, _| date_getter(&this, false, date::week_day));
    method(it, dp, "getHours", 0, |_, this, _| date_getter(&this, true, date::hour_from_time));
    method(it, dp, "getUTCHours", 0, |_, this, _| date_getter(&this, false, date::hour_from_time));
    method(it, dp, "getMinutes", 0, |_, this, _| date_getter(&this, true, date::min_from_time));
    method(it, dp, "getUTCMinutes", 0, |_, this, _| date_getter(&this, false, date::min_from_time));
    method(it, dp, "getSeconds", 0, |_, this, _| date_getter(&this, true, date::sec_from_time));
    method(it, dp, "getUTCSeconds", 0, |_, this, _| date_getter(&this, false, date::sec_from_time));
    method(it, dp, "getMilliseconds", 0, |_, this, _| date_getter(&this, true, date::ms_from_time));
    method(it, dp, "getUTCMilliseconds", 0, |_, this, _| date_getter(&this, false, date::ms_from_time));
    method(it, dp, "getTimezoneOffset", 0, |_, this, _| date_getter(&this, false, |t| (t - date::local_time(t)) / 60000.0));
    method(it, dp, "setTime", 1, |it, this, a| {
        let (o, _) = this_time(&this)?;
        let t = num_arg(it, a, 0)?;
        Ok(set_time(&o, date::time_clip(t)))
    });
    method(it, dp, "setMilliseconds", 1, |it, this, a| date_setter(it, this, a, true, 6, 1));
    method(it, dp, "setUTCMilliseconds", 1, |it, this, a| date_setter(it, this, a, false, 6, 1));
    method(it, dp, "setSeconds", 2, |it, this, a| date_setter(it, this, a, true, 5, 2));
    method(it, dp, "setUTCSeconds", 2, |it, this, a| date_setter(it, this, a, false, 5, 2));
    method(it, dp, "setMinutes", 3, |it, this, a| date_setter(it, this, a, true, 4, 3));
    method(it, dp, "setUTCMinutes", 3, |it, this, a| date_setter(it, this, a, false, 4, 3));
    method(it, dp, "setHours", 4, |it, this, a| date_setter(it, this, a, true, 3, 4));
    method(it, dp, "setUTCHours", 4, |it, this, a| date_setter(it, this, a, false, 3, 4));
    method(it, dp, "setDate", 1, |it, this, a| date_setter(it, this, a, true, 2, 1));
    method(it, dp, "setUTCDate", 1, |it, this, a| date_setter(it, this, a, false, 2, 1));
    method(it, dp, "setMonth", 2, |it, this, a| date_setter(it, this, a, true, 1, 2));
    method(it, dp, "setUTCMonth", 2, |it, this, a| date_setter(it, this, a, false, 1, 2));
    method(it, dp, "setFullYear", 3, |it, this, a| date_setter(it, this, a, true, 0, 3));
    method(it, dp, "setUTCFullYear", 3, |it, this, a| date_setter(it, this, a, false, 0, 3));
    method(it, dp, "setYear", 1, |it, this, a| {
        let (o, t) = this_time(&this)?;
        let y = num_arg(it, a, 0)?;
        if y.is_nan() {
            return Ok(set_time(&o, f64::NAN));
        }
        let t = if t.is_nan() { 0.0 } else { date::local_time(t) };
        let yi = to_integer(y);
        let y = if (0.0..=99.0).contains(&yi) { 1900.0 + yi } else { y };
        let d = date::make_day(y, date::month_from_time(t), date::date_from_time(t));
        let r = date::make_date(d, date::time_within_day(t));
        Ok(set_time(&o, date::time_clip(date::utc(r))))
    });
}
