import axios from 'axios';

// API route to handle Facebook authentication callback
export default async function handler(req, res) {
  const { code, state } = req.query;

  if (!code || !state) {
    return res.status(400).json({ error: 'Missing required parameters' });
  }

  try {
    // Forward the code and state to the backend
    const backendUrl = process.env.BACKEND_URL || 'http://localhost:8080';
    const response = await axios.get(`${backendUrl}/auth/facebook/callback`, {
      params: { code, state }
    });

    // Return the user data to the client
    return res.status(200).json(response.data);
  } catch (error) {
    console.error('Auth callback error:', error);
    return res.status(500).json({ 
      error: 'Authentication failed', 
      details: error.response?.data || error.message 
    });
  }
}