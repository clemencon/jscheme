// Values, objects and properties (ECMA-262 1st edition, chapter 8).

use std::cell::RefCell;
use std::collections::HashMap;
use std::hash::{BuildHasherDefault, Hasher};
use std::rc::{Rc, Weak};

use crate::ast::FuncDecl;
use crate::interp::{Interp, JsResult};

pub type JsStr = Rc<[u16]>;
pub type ObjRef = Rc<RefCell<Object>>;

pub fn js(s: &str) -> JsStr {
    s.encode_utf16().collect::<Vec<u16>>().into()
}

pub fn rs(s: &[u16]) -> String {
    String::from_utf16_lossy(s)
}

thread_local! {
    static INTERN: RefCell<HashMap<Vec<u16>, JsStr, Fx>> = RefCell::new(HashMap::default());
}

/// Returns the shared copy of a string used as a name, so that names compare by pointer.
pub fn intern(s: &[u16]) -> JsStr {
    INTERN.with(|m| {
        let mut m = m.borrow_mut();
        if let Some(v) = m.get(s) {
            return v.clone();
        }
        let r: JsStr = s.into();
        m.insert(s.to_vec(), r.clone());
        r
    })
}

pub fn intern_str(s: &str) -> JsStr {
    let v: Vec<u16> = s.encode_utf16().collect();
    intern(&v)
}

#[derive(Clone)]
#[repr(u64)]
pub enum Value {
    Undef,
    Null,
    Bool(bool),
    Num(f64),
    Str(JsStr),
    Obj(ObjRef),
}

impl Value {
    pub fn is_obj(&self) -> bool {
        matches!(self, Value::Obj(_))
    }
}

// Property attributes (§8.6.1).
pub const RO: u8 = 1;
pub const DE: u8 = 2;
pub const DD: u8 = 4;

#[derive(Clone)]
pub struct Prop {
    pub val: Value,
    pub attrs: u8,
}

#[derive(Default, Clone, Copy)]
pub struct FxHasher(u64);

const K: u64 = 0x51_7c_c1_b7_27_22_0a_95;

impl Hasher for FxHasher {
    fn write(&mut self, bytes: &[u8]) {
        let mut h = self.0;
        let mut chunks = bytes.chunks_exact(8);
        for c in &mut chunks {
            h = (h.rotate_left(5) ^ u64::from_le_bytes(c.try_into().unwrap())).wrapping_mul(K);
        }
        for &b in chunks.remainder() {
            h = (h.rotate_left(5) ^ b as u64).wrapping_mul(K);
        }
        self.0 = h;
    }
    fn write_usize(&mut self, n: usize) {
        self.0 = (self.0.rotate_left(5) ^ n as u64).wrapping_mul(K);
    }
    fn finish(&self) -> u64 {
        self.0
    }
}

pub type Fx = BuildHasherDefault<FxHasher>;

const SMALL: usize = 8;

/// An insertion-ordered property table.
#[derive(Default)]
pub struct PropMap {
    map: Option<HashMap<JsStr, u32, Fx>>,
    slots: Vec<Option<(JsStr, Prop)>>,
    holes: usize,
}

#[inline]
fn key_eq(a: &JsStr, b: &[u16]) -> bool {
    std::ptr::eq(a.as_ptr(), b.as_ptr()) && a.len() == b.len() || **a == *b
}

impl PropMap {
    fn index(&self, k: &[u16]) -> Option<usize> {
        match &self.map {
            Some(m) => m.get(k).map(|&i| i as usize),
            None => self.slots.iter().position(|s| matches!(s, Some((n, _)) if key_eq(n, k))),
        }
    }

    pub fn get(&self, k: &[u16]) -> Option<&Prop> {
        let i = self.index(k)?;
        self.slots[i].as_ref().map(|s| &s.1)
    }

    pub fn get_mut(&mut self, k: &[u16]) -> Option<&mut Prop> {
        let i = self.index(k)?;
        self.slots[i].as_mut().map(|s| &mut s.1)
    }

    /// The property in entry `i`, if that entry holds key `k`.
    #[inline]
    pub fn get_at(&self, i: usize, k: &[u16]) -> Option<&Prop> {
        match self.slots.get(i) {
            Some(Some((n, p))) if key_eq(n, k) => Some(p),
            _ => None,
        }
    }

    #[inline]
    pub fn get_at_mut(&mut self, i: usize, k: &[u16]) -> Option<&mut Prop> {
        match self.slots.get_mut(i) {
            Some(Some((n, p))) if key_eq(n, k) => Some(p),
            _ => None,
        }
    }

    pub fn contains(&self, k: &[u16]) -> bool {
        self.index(k).is_some()
    }

    /// Inserts or replaces the property (value and attributes).
    pub fn insert(&mut self, k: JsStr, p: Prop) {
        if let Some(i) = self.index(&k) {
            self.slots[i].as_mut().unwrap().1 = p;
            return;
        }
        let i = self.slots.len();
        if let Some(m) = &mut self.map {
            m.insert(k.clone(), i as u32);
        }
        self.slots.push(Some((k, p)));
        if self.map.is_none() && self.slots.len() > SMALL {
            self.rebuild();
        }
    }

    pub fn remove(&mut self, k: &[u16]) -> bool {
        let i = match self.index(k) {
            Some(i) => i,
            None => return false,
        };
        if let Some(m) = &mut self.map {
            m.remove(k);
        }
        self.slots[i] = None;
        self.holes += 1;
        if self.holes > 16 && self.holes * 2 > self.slots.len() {
            self.slots.retain(|s| s.is_some());
            self.holes = 0;
            if self.map.is_some() {
                self.rebuild();
            }
        }
        true
    }

    fn rebuild(&mut self) {
        let mut m: HashMap<JsStr, u32, Fx> = HashMap::default();
        m.reserve(self.slots.len());
        for (i, s) in self.slots.iter().enumerate() {
            if let Some((k, _)) = s {
                m.insert(k.clone(), i as u32);
            }
        }
        self.map = Some(m);
    }

    pub fn iter(&self) -> impl Iterator<Item = (&JsStr, &Prop)> {
        self.slots.iter().filter_map(|s| s.as_ref().map(|(k, p)| (k, p)))
    }
}

pub type NativeFn = fn(&mut Interp, Value, &[Value]) -> JsResult<Value>;

#[derive(Clone, Copy)]
pub struct Native {
    pub name: &'static str,
    pub call: NativeFn,
    pub ctor: Option<NativeFn>,
}

pub enum Func {
    User(Rc<FuncDecl>),
    Native(Native),
}

pub enum Kind {
    Plain,
    Activation,
    Array,
    /// The arguments object; index i is shared with parameter map[i] of the activation object.
    Arguments { act: Weak<RefCell<Object>>, map: Vec<Option<JsStr>> },
    Func(Func),
    Bool(bool),
    Num(f64),
    Str(JsStr),
    Date(f64),
}

pub struct Object {
    pub class: &'static str,
    pub proto: Option<ObjRef>,
    pub props: PropMap,
    pub kind: Kind,
}

impl Object {
    pub fn new(class: &'static str, proto: Option<ObjRef>, kind: Kind) -> ObjRef {
        Rc::new(RefCell::new(Object { class, proto, props: PropMap::default(), kind }))
    }

    pub fn set(&mut self, k: JsStr, v: Value, attrs: u8) {
        self.props.insert(k, Prop { val: v, attrs });
    }

    pub fn is_callable(&self) -> bool {
        matches!(self.kind, Kind::Func(_))
    }

    /// The mapped activation property for an index of an arguments object.
    fn arg_map(&self, k: &[u16]) -> Option<(ObjRef, JsStr)> {
        if let Kind::Arguments { act, map } = &self.kind {
            let i = array_index(k)? as usize;
            if let Some(Some(name)) = map.get(i) {
                if self.props.contains(k) {
                    return act.upgrade().map(|a| (a, name.clone()));
                }
            }
        }
        None
    }
}

/// [[Get]] walking the prototype chain; None if no object in the chain has the property.
pub fn lookup(o: &ObjRef, k: &[u16]) -> Option<Value> {
    let b = o.borrow();
    if let Some(p) = b.props.get(k) {
        if let Kind::Arguments { .. } = b.kind {
            if let Some((a, n)) = b.arg_map(k) {
                drop(b);
                return Some(get(&a, &n));
            }
        }
        return Some(p.val.clone());
    }
    let mut cur = b.proto.clone();
    drop(b);
    while let Some(o) = cur {
        let b = o.borrow();
        if let Some(p) = b.props.get(k) {
            if let Kind::Arguments { .. } = b.kind {
                if let Some((a, n)) = b.arg_map(k) {
                    drop(b);
                    return Some(get(&a, &n));
                }
            }
            return Some(p.val.clone());
        }
        cur = b.proto.clone();
    }
    None
}

pub fn get(o: &ObjRef, k: &[u16]) -> Value {
    lookup(o, k).unwrap_or(Value::Undef)
}

pub fn has_property(o: &ObjRef, k: &[u16]) -> bool {
    let mut cur = o.clone();
    loop {
        let next = {
            let b = cur.borrow();
            if b.props.contains(k) {
                return true;
            }
            match &b.proto {
                Some(p) => p.clone(),
                None => return false,
            }
        };
        cur = next;
    }
}

/// [[CanPut]] (§8.6.2.3).
pub fn can_put(o: &ObjRef, k: &[u16]) -> bool {
    let mut cur = o.clone();
    loop {
        let next = {
            let b = cur.borrow();
            if let Some(p) = b.props.get(k) {
                return p.attrs & RO == 0;
            }
            match &b.proto {
                Some(p) => p.clone(),
                None => return true,
            }
        };
        cur = next;
    }
}

/// [[Put]] for objects other than arrays (§8.6.2.2).
pub fn put_plain(o: &ObjRef, k: &JsStr, v: Value) {
    if !can_put(o, k) {
        return;
    }
    let mut b = o.borrow_mut();
    if let Some((a, n)) = b.arg_map(k) {
        if let Some(p) = b.props.get_mut(k) {
            p.val = v.clone();
        }
        drop(b);
        put_plain(&a, &n, v);
        return;
    }
    if let Some(p) = b.props.get_mut(k) {
        p.val = v;
    } else {
        b.props.insert(k.clone(), Prop { val: v, attrs: 0 });
    }
}

/// [[Delete]] (§8.6.2.5).
pub fn delete(o: &ObjRef, k: &[u16]) -> bool {
    let mut b = o.borrow_mut();
    match b.props.get(k) {
        None => true,
        Some(p) if p.attrs & DD != 0 => false,
        Some(_) => {
            if let Kind::Arguments { map, .. } = &mut b.kind {
                if let Some(i) = array_index(k) {
                    if let Some(m) = map.get_mut(i as usize) {
                        *m = None;
                    }
                }
            }
            b.props.remove(k);
            true
        }
    }
}

/// The value of P as an array index (§15.4), if it is one.
pub fn array_index(k: &[u16]) -> Option<u32> {
    if k.is_empty() || k.len() > 10 {
        return None;
    }
    if k[0] == b'0' as u16 && k.len() > 1 {
        return None;
    }
    let mut v: u64 = 0;
    for &c in k {
        if !(0x30..=0x39).contains(&c) {
            return None;
        }
        v = v * 10 + (c - 0x30) as u64;
    }
    if v >= 4294967295 {
        return None;
    }
    Some(v as u32)
}
