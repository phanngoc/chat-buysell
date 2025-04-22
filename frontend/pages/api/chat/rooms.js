import axios from 'axios';

export default async function handler(req, res) {
  const { userId } = req.query;
  
  if (!userId) {
    return res.status(400).json({ error: 'Missing user ID' });
  }

  try {
    const backendUrl = process.env.BACKEND_URL || 'http://localhost:8080';
    const response = await axios.get(`${backendUrl}/chat/rooms`, {
      params: { userId }
    });

    return res.status(200).json(response.data);
  } catch (error) {
    console.error('Error fetching chat rooms:', error);
    return res.status(error.response?.status || 500).json({
      error: 'Failed to fetch chat rooms',
      details: error.response?.data || error.message
    });
  }
}