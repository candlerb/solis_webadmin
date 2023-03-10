const el = (sel, par) => (par || document).querySelector(sel);
const els = (sel, par) => (par || document).querySelectorAll(sel);
const elNew = (tag, prop) => Object.assign(document.createElement(tag), prop);
const attr = (el, attr) => Object.entries(attr).forEach(([k, v]) => el.setAttribute(k, v));
const css = (el, styles) => Object.assign(el.style, styles);

function panic(e) {
    alert('Something went wrong! ' + e.name + ': ' + e.message);
}

function mkpanic(spinner) {
    return function(e) {
        spinner.style.visibility="hidden";
        panic(e);
    }
}

function get_om() {
    const spinner = document.getElementById("loading_om");
    spinner.style.visibility="visible";
    read_registers(43110, 1)
    .then(function(r) {
        spinner.style.visibility="hidden";
        v = r[0];
        options = [
            elNew("option", {
                value: 33,
                text: "Stop",
                selected: v == 33,
            }),
            elNew("option", {
                value: 35,
                text: "Run",
                selected: v == 35,
            }),
        ];
        if (!options.find(e => e.selected)) {
            options.push(
                elNew("option", {
                    value: r,
                    text: r,
                    selected: true,
                })
            );
        }
        document.getElementById("run_mode").replaceChildren(...options);
    })
    .catch(mkpanic(spinner));
}

function set_om() {
    v = document.getElementById("run_mode").value;
    if (!v) { return; }
    const spinner = document.getElementById("loading_om");
    spinner.style.visibility="visible";
    write_register(43110, parseInt(v))
    .then(function() {
        spinner.style.visibility="hidden";
        alert('Done!');
    })
    .catch(mkpanic(spinner));
}

function get_charge() {
    const spinner = document.getElementById("loading_charge");
    spinner.style.visibility="visible";
    read_registers(43117, 1)
    .then(function(r) {
        spinner.style.visibility="hidden";
        v = r[0];
        options = [
            elNew("option", {
                value: 10,
                text: "1A",
                selected: v == 10,
            }),
            elNew("option", {
                value: 50,
                text: "5A",
                selected: v == 50,
            }),
            elNew("option", {
                value: 1000,
                text: "100A",
                selected: v == 1000,
            }),
        ];
        if (!options.find(e => e.selected)) {
            options.push(
                elNew("option", {
                    value: r,
                    text: r/10+"A",
                    selected: true,
                })
            );
        }
        document.getElementById("charge_limit").replaceChildren(...options);
    })
    .catch(mkpanic(spinner));
}

function set_charge() {
    v = document.getElementById("charge_limit").value;
    if (!v) { return; }
    const spinner = document.getElementById("loading_charge");
    spinner.style.visibility="visible";
    write_register(43117, parseInt(v))
    .then(function() {
        spinner.style.visibility="hidden";
        alert('Done!');
    })
    .catch(mkpanic(spinner));
}

function get_discharge() {
    const spinner = document.getElementById("loading_discharge");
    spinner.style.visibility="visible";
    read_registers(43118, 1)
    .then(function(r) {
        spinner.style.visibility="hidden";
        v = r[0];
        options = [
            elNew("option", {
                value: 10,
                text: "1A",
                selected: v == 10,
            }),
            elNew("option", {
                value: 50,
                text: "5A",
                selected: v == 50,
            }),
            elNew("option", {
                value: 1000,
                text: "100A",
                selected: v == 1000,
            }),
        ];
        if (!options.find(e => e.selected)) {
            options.push(
                elNew("option", {
                    value: r,
                    text: r/10+"A",
                    selected: true,
                })
            );
        }
        document.getElementById("discharge_limit").replaceChildren(...options);
    })
    .catch(mkpanic(spinner));
}

function set_discharge() {
    v = document.getElementById("discharge_limit").value;
    if (!v) { return; }
    const spinner = document.getElementById("loading_discharge");
    spinner.style.visibility="visible";
    write_register(43118, parseInt(v))
    .then(function() {
        spinner.style.visibility="hidden";
        alert('Done!');
    })
    .catch(mkpanic(spinner));
}

function mktime(h, m) {
    return h.toString().padStart(2, '0') + ':' + m.toString().padStart(2, '0');
}

function rdtime(s) {
    if (!s) { return [0, 0] };
    const regex1 = RegExp('^([0-9]+):([0-9]+)$');
    let res = regex1.exec(s);
    if (res) {
        return [parseInt(res[1]), parseInt(res[2])];
    }
    throw "Invalid time: '" + s + "'";
}

function get_times() {
    const spinner = document.getElementById("loading_times");
    spinner.style.visibility="visible";
    read_registers(43143, 28)
    .then(function(r) {
        spinner.style.visibility="hidden";
        document.getElementById("charge_u0").value = mktime(r[0], r[1]);
        document.getElementById("charge_v0").value = mktime(r[2], r[3]);
        document.getElementById("discharge_u0").value = mktime(r[4], r[5]);
        document.getElementById("discharge_v0").value = mktime(r[6], r[7]);
        document.getElementById("charge_u1").value = mktime(r[10], r[11]);
        document.getElementById("charge_v1").value = mktime(r[12], r[13]);
        document.getElementById("discharge_u1").value = mktime(r[14], r[15]);
        document.getElementById("discharge_v1").value = mktime(r[16], r[17]);
        document.getElementById("charge_u2").value = mktime(r[20], r[21]);
        document.getElementById("charge_v2").value = mktime(r[22], r[23]);
        document.getElementById("discharge_u2").value = mktime(r[24], r[25]);
        document.getElementById("discharge_v2").value = mktime(r[26], r[27]);
    })
    .catch(mkpanic(spinner));
}

function set_times() {
    vals = [].concat(
        rdtime(document.getElementById("charge_u0").value),
        rdtime(document.getElementById("charge_v0").value),
        rdtime(document.getElementById("discharge_u0").value),
        rdtime(document.getElementById("discharge_v0").value),
        [0, 0],
        rdtime(document.getElementById("charge_u1").value),
        rdtime(document.getElementById("charge_v1").value),
        rdtime(document.getElementById("discharge_u1").value),
        rdtime(document.getElementById("discharge_v1").value),
        [0, 0],
        rdtime(document.getElementById("charge_u2").value),
        rdtime(document.getElementById("charge_v2").value),
        rdtime(document.getElementById("discharge_u2").value),
        rdtime(document.getElementById("discharge_v2").value),
    );
    const spinner = document.getElementById("loading_times");
    spinner.style.visibility="visible";
    write_registers(43143, vals)
    .then(function() {
        spinner.style.visibility="hidden";
        alert('Done!');
    })
    .catch(mkpanic(spinner));
}

function get_rates() {
    const spinner = document.getElementById("loading_rates");
    spinner.style.visibility="visible";
    read_registers(43141, 2)
    .then(function(r) {
        spinner.style.visibility="hidden";
        document.getElementById("timed_charge_rate").setAttribute("value", r[0]);
        document.getElementById("timed_charge_rate").nextElementSibling.value = r[0]/10;
        document.getElementById("timed_discharge_rate").setAttribute("value", r[1]);
        document.getElementById("timed_discharge_rate").nextElementSibling.value = r[1]/10;
    })
    .catch(mkpanic(spinner));
}

function set_rates() {
    v1 = document.getElementById("timed_charge_rate").value;
    v2 = document.getElementById("timed_discharge_rate").value;
    if (!v1 || !v2 || v1 < 1 || v2 < 1 || v1 > 700 || v2 > 700) { return; }
    const spinner = document.getElementById("loading_rates");
    spinner.style.visibility="visible";
    write_registers(43141, [v1, v2])
    .then(function() {
        spinner.style.visibility="hidden";
        alert('Done!');
    })
    .catch(mkpanic(spinner));
}
