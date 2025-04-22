import { useRouter } from 'next/router';
import { useEffect, useState } from 'react';
import axios from 'axios';
import { useAuth } from '../../components/AuthContext';

export default function Callback() {
  const router = useRouter();
  const { setUserData } = useAuth();
  const [error, setError] = useState(null);

  useEffect(() => {
    const handleCallback = async () => {
      // Get the callback data from query params
      const { code, state } = router.query;

      if (!code || !state) {
        return; // Wait for the query params to be populated
      }

      try {
        // Exchange the code for user data
        const response = await axios.get(`/api/auth/callback?code=${code}&state=${state}`);
        const userData = response.data.user;

        // Save the user data to context
        setUserData(userData);
        
        // Redirect to chat page
        router.push('/chat');
      } catch (err) {
        console.error('Authentication error:', err);
        setError('Failed to authenticate with Facebook. Please try again.');
      }
    };

    if (router.isReady) {
      handleCallback();
    }
  }, [router.isReady, router.query]);

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gray-50">
      <div className="bg-white p-8 rounded-lg shadow-md text-center">
        {error ? (
          <div>
            <h2 className="text-2xl font-bold text-red-500 mb-4">Authentication Error</h2>
            <p className="text-gray-600 mb-4">{error}</p>
            <button 
              onClick={() => router.push('/')}
              className="btn-primary"
            >
              Return to Homepage
            </button>
          </div>
        ) : (
          <div>
            <h2 className="text-2xl font-bold text-gray-800 mb-4">Logging in...</h2>
            <div className="flex justify-center">
              <div className="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-primary"></div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}