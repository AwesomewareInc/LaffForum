onmessage = (e) => {
	fetch(e.data)
		.then(t => t.text())
		.then(t => {
			postMessage(t);
			console.log("fetched "+e.data);
		});
};