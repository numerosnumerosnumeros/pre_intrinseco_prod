const REQUESTS = 10;

function getCSRFToken() {
	const csrfTokenCookie = document.cookie
		.split('; ')
		.find((row) => row.startsWith('nodo_csrf_token='));
	if (!csrfTokenCookie) {
		return null;
	}
	return csrfTokenCookie.split('=')[1];
}

async function submitFinanceData() {
	const formData = new FormData();

	formData.append('ticker', 'TESLA');
	formData.append('minpag', '1');
	formData.append('maxpag', '1');
	formData.append('isurl', 'true');
	formData.append('period', '2024-Q3');
	formData.append('currency', 'USD');
	formData.append(
		'urlvalue',
		'https://www.sec.gov/Archives/edgar/data/1318605/000162828024043486/tsla-20240930.htm'
	);

	try {
		const response = await fetch('/api/finances/submit', {
			method: 'POST',
			credentials: 'include',
			headers: {
				Accept: 'application/json',
				'X-CSRF-Token': getCSRFToken(),
			},
			body: formData,
		});

		return { status: response.status };
	} catch (error) {
		console.error('Error submitting data:', error);
		return { status: 'error', error: error.message };
	}
}

async function makeMultipleRequests() {
	console.log(`Starting ${REQUESTS} simultaneous requests...`);

	const startTime = Date.now();
	const promises = Array(REQUESTS)
		.fill()
		.map(() => submitFinanceData());

	try {
		const results = await Promise.all(promises);
		const endTime = Date.now();

		const statusCounts = results.reduce((counts, result) => {
			counts[result.status] = (counts[result.status] || 0) + 1;
			return counts;
		}, {});

		console.log(
			`Completed ${REQUESTS} requests in ${
				(endTime - startTime) / 1000
			} seconds`
		);
		console.log('Status code counts:', statusCounts);

		return results;
	} catch (error) {
		console.error('Failed to complete all requests:', error);
	}
}

makeMultipleRequests();
