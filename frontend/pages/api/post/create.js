import axios from 'axios';

export default async function handler(req, res) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const { userId, content, type } = req.body;
  
  if (!userId || !content || !type) {
    return res.status(400).json({ error: 'Missing required fields' });
  }

  try {
    const backendUrl = process.env.BACKEND_URL || 'http://localhost:8080';
    const response = await axios.post(`${backendUrl}/post/create`, {
      userId,
      content,
      type
    });

    return res.status(200).json(response.data);
  } catch (error) {
    console.error('Error creating post:', error);
    return res.status(error.response?.status || 500).json({
      error: 'Failed to create post',
      details: error.response?.data || error.message
    });
  }
}