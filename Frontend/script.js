document.addEventListener('DOMContentLoaded', () => {
    const signupForm = document.getElementById('signupForm');
    const loginForm = document.getElementById('loginForm');
    const showLogin = document.getElementById('showLogin');
    const showSignup = document.getElementById('showSignup');
    const errorMessage = document.getElementById('errorMessage');

    // URLs for your backend endpoints
    const SIGNUP_URL = 'http://localhost:8080/api/signup';
    const LOGIN_URL = 'http://localhost:8080/api/login';

    // --- Event Listeners to Toggle Forms ---
    showLogin.addEventListener('click', (e) => {
        e.preventDefault();
        loginForm.classList.remove('hidden');
        signupForm.classList.add('hidden');
        errorMessage.textContent = '';
    });

    showSignup.addEventListener('click', (e) => {
        e.preventDefault();
        signupForm.classList.remove('hidden');
        loginForm.classList.add('hidden');
        errorMessage.textContent = '';
    });

    // --- Signup Form Submission ---
    signupForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        errorMessage.textContent = '';

        const username = document.getElementById('signupUsername').value;
        const email = document.getElementById('signupEmail').value;
        const password = document.getElementById('signupPassword').value;

        try {
            const response = await fetch(SIGNUP_URL, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ username, email, password }),
            });

            if (response.ok) {
                // Successful signup: log in the user and redirect
                // Note: In a real app, the server would return a session token here.
                localStorage.setItem('currentUser', username); // *** NEW: Store username ***
                window.location.href = 'welcome.html';
            } else {
                // If there's an error, display it
                const errorText = await response.text();
                errorMessage.textContent = errorText;
            }
        } catch (error) {
            errorMessage.textContent = 'Network error. Could not connect to the server.';
        }
    });

    // --- Login Form Submission ---
    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        errorMessage.textContent = '';

        const username = document.getElementById('loginUsername').value;
        const password = document.getElementById('loginPassword').value;

        try {
            const response = await fetch(LOGIN_URL, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ username, password }),
            });

            if (response.ok) {
                // If login is successful, store username and redirect
                localStorage.setItem('currentUser', username); // *** NEW: Store username ***
                window.location.href = 'welcome.html';
            } else {
                // If there's an error, display it
                const errorText = await response.text();
                errorMessage.textContent = errorText;
            }
        } catch (error) {
            errorMessage.textContent = 'Network error. Could not connect to the server.';
        }
    });
});