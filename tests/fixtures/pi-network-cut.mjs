// AI-048 (#1810): cut the network inside a Node process.
//
// Loaded with NODE_OPTIONS=--import=<this file>. Every outbound attempt the
// process can make -- the global fetch, net.Socket#connect (which http, https
// and undici all reach) and tls.connect -- throws instead of dialling, so the
// process runs as if the machine were unplugged. A proxy variable cannot do
// this: Node's fetch ignores HTTPS_PROXY unless the caller opts in, and
// pi-nan-provider does not.
//
// On load it writes PI_NETWORK_CUT_MARKER. A test that cuts the network must
// assert the marker exists, or a dropped NODE_OPTIONS leaves the process
// online and the test green for the wrong reason. Each refused attempt is
// appended to the marker, so the test can also say what tried to dial.
import fs from "node:fs";
import net from "node:net";
import tls from "node:tls";

const marker = process.env.PI_NETWORK_CUT_MARKER;
if (!marker) {
	throw new Error("pi-network-cut: PI_NETWORK_CUT_MARKER is unset");
}
fs.writeFileSync(marker, `loaded pid=${process.pid}\n`);

const refuse = (what) => {
	fs.appendFileSync(marker, `refused ${what}\n`);
	const error = new Error(`network cut by tests/fixtures/pi-network-cut.mjs: ${what}`);
	error.code = "ENETUNREACH";
	throw error;
};

globalThis.fetch = (input) => Promise.reject(refuse(`fetch ${typeof input === "string" ? input : (input?.url ?? String(input))}`));

net.Socket.prototype.connect = function (...args) {
	const options = typeof args[0] === "object" && args[0] !== null ? args[0] : { host: args[1], port: args[0] };
	refuse(`net.connect ${options.host ?? ""}:${options.port ?? options.path ?? ""}`);
};

tls.connect = (...args) => {
	const options = typeof args[0] === "object" && args[0] !== null ? args[0] : { host: args[1], port: args[0] };
	refuse(`tls.connect ${options.host ?? ""}:${options.port ?? ""}`);
};
