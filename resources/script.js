window.onpopstate = (e) => {
	window.location.reload();
};

let pages = new Map();
var parser = new DOMParser();

function replaceLinks() {
	let links = document.querySelectorAll('a');
	// For each link on the page.
	links.forEach((link =>  {
		// If it's not from us, don't bother.
		if(!link.href.includes(window.location.host)) {
			return
		}
		// DON'T cache the account pages.
		if(link.href.includes("logout") || link.href.includes("login") || link.href.includes("register")) {
			return
		}

		// Check if we already have its contents cached
		let dest = pages.get(link.href);

		// If we do, set the link to display those cached contents.
		if(dest != undefined) {
			link.addEventListener("click", e => {
				e.preventDefault();
				history.pushState({}, link.title, link.href);
				var doc = parser.parseFromString(dest, 'text/html');
				document.body.innerHTML = doc.body.innerHTML;
				replaceLinks();
				return false;
			})
			return
		}

		// Otherwise, start a new thread to download the page this links to.
		let newWorker = new Worker('/resources/prefetchworker.js');
		newWorker.postMessage(link.href);

		// When it's finished, add the contents to the map and set
		// the link to use it.
		newWorker.onmessage = (e) => {
			pages.set(link.href, e.data);
			// when the link is clicked...
			link.addEventListener("click", f => {
				f.preventDefault();
				history.pushState({}, link.title, link.href);
				var doc = parser.parseFromString(e.data, 'text/html');
				document.body.innerHTML = doc.body.innerHTML;
				replaceLinks();
				return false;
			})
		}
	}))
}
replaceLinks();