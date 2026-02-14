// BicicletAPP - Frontend JavaScript
// Minimal JS for enhanced UX

document.addEventListener('DOMContentLoaded', function () {
    // Auto-hide flash messages after 5 seconds
    const flashMessages = document.querySelectorAll('.flash');
    flashMessages.forEach(flash => {
        setTimeout(() => {
            flash.style.transition = 'opacity 0.5s';
            flash.style.opacity = '0';
            setTimeout(() => flash.remove(), 500);
        }, 5000);
    });

    // Form confirmation for destructive actions
    document.querySelectorAll('form[data-confirm]').forEach(form => {
        form.addEventListener('submit', function (e) {
            if (!confirm(this.dataset.confirm)) {
                e.preventDefault();
            }
        });
    });

    // Auto-refresh tracking status every 30 seconds
    const trackingStatus = document.querySelector('.tracking-result');
    if (trackingStatus) {
        setInterval(() => {
            location.reload();
        }, 30000);
    }

    // Date picker min date = today
    const datePickers = document.querySelectorAll('input[type="date"]');
    const today = new Date().toISOString().split('T')[0];
    datePickers.forEach(picker => {
        if (!picker.min) {
            picker.min = today;
        }
    });

    // Password confirmation validation
    const confirmPassword = document.getElementById('confirm_password');
    const password = document.getElementById('password');
    if (confirmPassword && password) {
        confirmPassword.addEventListener('input', function () {
            if (this.value !== password.value) {
                this.setCustomValidity('Las contraseñas no coinciden');
            } else {
                this.setCustomValidity('');
            }
        });
    }

    // Mobile menu toggle
    const menuToggle = document.querySelector('.menu-toggle');
    const nav = document.querySelector('nav ul:last-child');
    if (menuToggle && nav) {
        menuToggle.addEventListener('click', () => {
            nav.classList.toggle('show');
        });
    }
});

// Utility functions

// Format money in local currency
function formatMoney(amount) {
    return new Intl.NumberFormat('es-AR', {
        style: 'currency',
        currency: 'ARS'
    }).format(amount);
}

// Format date in local format
function formatDate(dateString) {
    return new Date(dateString).toLocaleDateString('es-AR');
}

// Debounce function for search inputs
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// API helper
async function api(endpoint, options = {}) {
    const response = await fetch(endpoint, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...options.headers
        }
    });

    if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
    }

    return response.json();
}
