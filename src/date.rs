// Date arithmetic (§15.9.1), formatting and parsing.

pub const MS_PER_DAY: f64 = 86400000.0;
const MS_PER_HOUR: f64 = 3600000.0;
const MS_PER_MINUTE: f64 = 60000.0;
const MS_PER_SECOND: f64 = 1000.0;

fn modulo(x: f64, y: f64) -> f64 {
    let r = x % y;
    if r < 0.0 { r + y } else { r }
}

pub fn day(t: f64) -> f64 {
    (t / MS_PER_DAY).floor()
}

pub fn time_within_day(t: f64) -> f64 {
    modulo(t, MS_PER_DAY)
}

fn days_in_year(y: f64) -> f64 {
    if y % 4.0 != 0.0 {
        365.0
    } else if y % 100.0 != 0.0 {
        366.0
    } else if y % 400.0 != 0.0 {
        365.0
    } else {
        366.0
    }
}

pub fn day_from_year(y: f64) -> f64 {
    365.0 * (y - 1970.0) + ((y - 1969.0) / 4.0).floor() - ((y - 1901.0) / 100.0).floor() + ((y - 1601.0) / 400.0).floor()
}

fn time_from_year(y: f64) -> f64 {
    MS_PER_DAY * day_from_year(y)
}

pub fn year_from_time(t: f64) -> f64 {
    let mut y = (t / (365.2425 * MS_PER_DAY)).floor() + 1970.0;
    while time_from_year(y) > t {
        y -= 1.0;
    }
    while time_from_year(y + 1.0) <= t {
        y += 1.0;
    }
    y
}

fn in_leap_year(t: f64) -> f64 {
    if days_in_year(year_from_time(t)) == 366.0 { 1.0 } else { 0.0 }
}

fn day_within_year(t: f64) -> f64 {
    day(t) - day_from_year(year_from_time(t))
}

/// First day within the year of each month, for a non-leap year.
const MONTH_START: [f64; 13] = [0.0, 31.0, 59.0, 90.0, 120.0, 151.0, 181.0, 212.0, 243.0, 273.0, 304.0, 334.0, 365.0];

fn month_start(m: usize, leap: f64) -> f64 {
    MONTH_START[m] + if m >= 2 { leap } else { 0.0 }
}

pub fn month_from_time(t: f64) -> f64 {
    let d = day_within_year(t);
    let leap = in_leap_year(t);
    for m in 0..12 {
        if d < month_start(m + 1, leap) {
            return m as f64;
        }
    }
    11.0
}

pub fn date_from_time(t: f64) -> f64 {
    let d = day_within_year(t);
    let leap = in_leap_year(t);
    let m = month_from_time(t) as usize;
    d - month_start(m, leap) + 1.0
}

pub fn week_day(t: f64) -> f64 {
    modulo(day(t) + 4.0, 7.0)
}

pub fn hour_from_time(t: f64) -> f64 {
    modulo((t / MS_PER_HOUR).floor(), 24.0)
}

pub fn min_from_time(t: f64) -> f64 {
    modulo((t / MS_PER_MINUTE).floor(), 60.0)
}

pub fn sec_from_time(t: f64) -> f64 {
    modulo((t / MS_PER_SECOND).floor(), 60.0)
}

pub fn ms_from_time(t: f64) -> f64 {
    modulo(t, MS_PER_SECOND)
}

fn to_integer(n: f64) -> f64 {
    crate::interp::to_integer(n)
}

pub fn make_time(h: f64, m: f64, s: f64, ms: f64) -> f64 {
    if !h.is_finite() || !m.is_finite() || !s.is_finite() || !ms.is_finite() {
        return f64::NAN;
    }
    to_integer(h) * MS_PER_HOUR + to_integer(m) * MS_PER_MINUTE + to_integer(s) * MS_PER_SECOND + to_integer(ms)
}

pub fn make_day(y: f64, m: f64, d: f64) -> f64 {
    if !y.is_finite() || !m.is_finite() || !d.is_finite() {
        return f64::NAN;
    }
    let (y, m, d) = (to_integer(y), to_integer(m), to_integer(d));
    let ym = y + (m / 12.0).floor();
    let mn = modulo(m, 12.0);
    if ym.abs() > 400000.0 {
        return f64::NAN;
    }
    let leap = if days_in_year(ym) == 366.0 { 1.0 } else { 0.0 };
    day_from_year(ym) + month_start(mn as usize, leap) + d - 1.0
}

pub fn make_date(day: f64, time: f64) -> f64 {
    if !day.is_finite() || !time.is_finite() {
        return f64::NAN;
    }
    day * MS_PER_DAY + time
}

pub fn time_clip(t: f64) -> f64 {
    if !t.is_finite() || t.abs() > 8.64e15 {
        return f64::NAN;
    }
    to_integer(t) + 0.0
}

/// The local time zone adjustment: the host contract runs with TZ=UTC.
pub fn local_time(t: f64) -> f64 {
    t
}

pub fn utc(t: f64) -> f64 {
    t
}

const DAYS: [&str; 7] = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const MONTHS: [&str; 12] = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

fn year_str(y: f64) -> String {
    if y < 0.0 { format!("-{:04}", -y as i64) } else { format!("{:04}", y as i64) }
}

/// Date.prototype.toString: "Fri Jun 15 2001 10:20:30 GMT+0000".
pub fn to_string(t: f64) -> String {
    if t.is_nan() {
        return "Invalid Date".into();
    }
    let l = local_time(t);
    format!(
        "{} {} {:02} {} {:02}:{:02}:{:02} GMT+0000",
        DAYS[week_day(l) as usize],
        MONTHS[month_from_time(l) as usize],
        date_from_time(l) as i64,
        year_str(year_from_time(l)),
        hour_from_time(l) as i64,
        min_from_time(l) as i64,
        sec_from_time(l) as i64
    )
}

/// Date.prototype.toUTCString: "Fri, 15 Jun 2001 10:20:30 GMT".
pub fn to_utc_string(t: f64) -> String {
    if t.is_nan() {
        return "Invalid Date".into();
    }
    format!(
        "{}, {:02} {} {} {:02}:{:02}:{:02} GMT",
        DAYS[week_day(t) as usize],
        date_from_time(t) as i64,
        MONTHS[month_from_time(t) as usize],
        year_str(year_from_time(t)),
        hour_from_time(t) as i64,
        min_from_time(t) as i64,
        sec_from_time(t) as i64
    )
}

#[derive(Debug, PartialEq)]
enum DTok {
    Num(f64, usize),
    Word(String),
    Sym(char),
}

/// Date.parse (§15.9.4.2): reads the formats produced by toString and toUTCString, and
/// common variations of them. Returns NaN for a string it cannot read.
pub fn parse(s: &str) -> f64 {
    let cs: Vec<char> = s.chars().collect();
    let mut toks = Vec::new();
    let mut i = 0;
    while i < cs.len() {
        let c = cs[i];
        if c.is_ascii_digit() {
            let st = i;
            while i < cs.len() && cs[i].is_ascii_digit() {
                i += 1;
            }
            let t: String = cs[st..i].iter().collect();
            toks.push(DTok::Num(t.parse().unwrap_or(f64::NAN), i - st));
        } else if c.is_ascii_alphabetic() {
            let st = i;
            while i < cs.len() && cs[i].is_ascii_alphabetic() {
                i += 1;
            }
            toks.push(DTok::Word(cs[st..i].iter().collect::<String>().to_ascii_lowercase()));
        } else if c.is_whitespace() || c == ',' {
            i += 1;
        } else if c == '(' {
            while i < cs.len() && cs[i] != ')' {
                i += 1;
            }
            i += 1;
        } else {
            toks.push(DTok::Sym(c));
            i += 1;
        }
    }
    let mut month: Option<f64> = None;
    let mut nums: Vec<f64> = Vec::new();
    let mut time = [0.0f64; 4];
    let mut have_time = false;
    let mut offset: f64 = 0.0;
    let mut tz_seen = false;
    let mut pm: Option<bool> = None;
    let mut slash_date: Vec<f64> = Vec::new();
    let mut k = 0;
    while k < toks.len() {
        match &toks[k] {
            DTok::Word(w) => {
                if let Some(m) = MONTHS.iter().position(|m| w.len() >= 3 && m.to_ascii_lowercase() == w[..3]) {
                    month = Some(m as f64);
                } else if DAYS.iter().any(|d| w.len() >= 3 && d.to_ascii_lowercase() == w[..3]) {
                } else if w == "gmt" || w == "utc" || w == "ut" || w == "z" {
                    tz_seen = true;
                } else if w == "am" || w == "pm" {
                    pm = Some(w == "pm");
                } else if w == "t" {
                } else {
                    return f64::NAN;
                }
                k += 1;
            }
            DTok::Num(n, _) => {
                let n = *n;
                if matches!(toks.get(k + 1), Some(DTok::Sym(':'))) && !have_time {
                    have_time = true;
                    time[0] = n;
                    k += 2;
                    let mut idx = 1;
                    while let Some(DTok::Num(v, _)) = toks.get(k) {
                        time[idx] = *v;
                        k += 1;
                        if idx < 2 && matches!(toks.get(k), Some(DTok::Sym(':'))) {
                            idx += 1;
                            k += 1;
                        } else if idx == 2 && matches!(toks.get(k), Some(DTok::Sym('.'))) {
                            idx = 3;
                            k += 1;
                        } else {
                            break;
                        }
                    }
                } else if matches!(toks.get(k + 1), Some(DTok::Sym('/'))) || (!slash_date.is_empty() && slash_date.len() < 3) {
                    slash_date.push(n);
                    k += 1;
                    if matches!(toks.get(k), Some(DTok::Sym('/'))) {
                        k += 1;
                    }
                } else {
                    nums.push(n);
                    k += 1;
                }
            }
            DTok::Sym(c) => {
                let c = *c;
                if (c == '+' || c == '-') && (tz_seen || have_time) {
                    if let Some(DTok::Num(v, len)) = toks.get(k + 1) {
                        let (h, m) = if *len <= 2 {
                            let m = if matches!(toks.get(k + 2), Some(DTok::Sym(':'))) {
                                if let Some(DTok::Num(mm, _)) = toks.get(k + 3) { *mm } else { 0.0 }
                            } else {
                                0.0
                            };
                            (*v, m)
                        } else {
                            ((*v / 100.0).floor(), *v % 100.0)
                        };
                        let sign = if c == '+' { 1.0 } else { -1.0 };
                        offset = sign * (h * 60.0 + m);
                        tz_seen = true;
                        k += 2;
                        if matches!(toks.get(k), Some(DTok::Sym(':'))) {
                            k += 2;
                        }
                        continue;
                    }
                    return f64::NAN;
                } else if c == '-' && !have_time {
                    if let Some(DTok::Num(v, _)) = toks.get(k + 1) {
                        nums.push(-*v);
                        k += 2;
                        continue;
                    }
                    return f64::NAN;
                } else if c == '.' || c == '-' {
                    k += 1;
                } else {
                    return f64::NAN;
                }
            }
        }
    }
    let (y, m, d) = if let Some(m) = month {
        match nums.len() {
            2 => {
                if nums[0].abs() > 31.0 || nums[0] < 1.0 { (nums[0], m, nums[1]) } else { (nums[1], m, nums[0]) }
            }
            _ => return f64::NAN,
        }
    } else if slash_date.len() == 3 && nums.is_empty() {
        if slash_date[0] > 31.0 {
            (slash_date[0], slash_date[1] - 1.0, slash_date[2])
        } else {
            (slash_date[2], slash_date[0] - 1.0, slash_date[1])
        }
    } else {
        return f64::NAN;
    };
    let mut h = time[0];
    if let Some(p) = pm {
        if h == 12.0 {
            h = 0.0;
        }
        if p {
            h += 12.0;
        }
    }
    let ms = time[3];
    let t = make_date(make_day(y, m, d), make_time(h, time[1], time[2], ms));
    let t = if tz_seen { t - offset * MS_PER_MINUTE } else { utc(t) };
    time_clip(t)
}
