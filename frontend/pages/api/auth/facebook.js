// API route to handle Facebook authentication
export default function handler(req, res) {
  // In a production app, we would set proper CORS headers and validate the request

  // Redirect to the backend Facebook OAuth endpoint
  const backendUrl = process.env.BACKEND_URL || 'http://localhost:8080';
  res.redirect(302, `${backendUrl}/auth/facebook`);
}