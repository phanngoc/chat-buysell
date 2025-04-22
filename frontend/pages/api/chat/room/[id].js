import axios from 'axios';

export default async function handler(req, res) {
  const { id } = req.query;
  
  if (!id) {
    return res.status(400).json({ error: 'Missing room ID' });
  }

  try {
    const backendUrl = process.env.BACKEND_URL || 'http://localhost:8080';
    const response = await axios.get(`${backendUrl}/chat/room/${id}`);

    return res.status(200).json(response.data);
  } catch (error) {
    console.error(`Error fetching chat room ${id}:`, error);
    return res.status(error.response?.status || 500).json({
      error: 'Failed to fetch chat room',
      details: error.response?.data || error.message
    });
  }
}