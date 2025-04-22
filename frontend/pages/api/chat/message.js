import axios from 'axios';

export default async function handler(req, res) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const { roomId, senderId, content } = req.body;
  
  if (!roomId || !senderId || !content) {
    return res.status(400).json({ error: 'Missing required fields' });
  }

  try {
    const backendUrl = process.env.BACKEND_URL || 'http://localhost:8080';
    const response = await axios.post(`${backendUrl}/chat/message`, {
      roomId,
      senderId,
      content
    });

    return res.status(200).json(response.data);
  } catch (error) {
    console.error('Error sending message:', error);
    return res.status(error.response?.status || 500).json({
      error: 'Failed to send message',
      details: error.response?.data || error.message
    });
  }
}