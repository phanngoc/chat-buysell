import { useState } from 'react';
import axios from 'axios';
import { useRouter } from 'next/router';
import { useAuth } from './AuthContext';

export default function SearchResults({ results, onClose, onChatRoomCreated }) {
  const { user } = useAuth();
  const [isLoading, setIsLoading] = useState(false);
  const [loadingItemId, setLoadingItemId] = useState(null);
  const router = useRouter();

  // Create a chat room between buyer and seller
  const createChatRoom = async (result) => {
    if (!user) {
      router.push('/');
      return;
    }

    try {
      setIsLoading(true);
      setLoadingItemId(result.post.id);
      
      // Determine if the current user is buyer or seller
      const isBuyer = user.type === 'buyer' || result.post.type === 'ban';
      
      const response = await axios.post('/api/chat/room/create', {
        buyerId: isBuyer ? user.id : result.user.id,
        sellerId: isBuyer ? result.user.id : user.id,
        postId: result.post.id
      });

      // If successful, add the new room to the chat rooms list and open it
      if (response.data && response.data.chatRoom) {
        if (onChatRoomCreated) {
          onChatRoomCreated(response.data.chatRoom);
        }
        onClose();
      }
    } catch (error) {
      console.error('Error creating chat room:', error);
      alert('Failed to create chat room. Please try again.');
    } finally {
      setIsLoading(false);
      setLoadingItemId(null);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow-xl overflow-hidden">
      <div className="p-4 border-b border-gray-200 flex justify-between items-center">
        <h3 className="text-lg font-medium">Matching Results</h3>
        <button 
          onClick={onClose}
          className="text-gray-500 hover:text-gray-700"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
      
      <div className="max-h-96 overflow-y-auto">
        {results.length === 0 ? (
          <div className="p-6 text-center text-gray-500">
            No matching results found
          </div>
        ) : (
          <div>
            {results.map((result) => (
              <div 
                key={result.post.id}
                onClick={() => createChatRoom(result)}
                className={`p-4 border-b border-gray-100 hover:bg-gray-50 cursor-pointer ${loadingItemId === result.post.id ? 'opacity-70' : ''}`}
              >
                <div className="flex items-start">
                  {result.user.avatar && (
                    <img 
                      src={result.user.avatar} 
                      alt={result.user.username || 'User'} 
                      className="w-10 h-10 rounded-full mr-3"
                    />
                  )}
                  <div className="flex-1">
                    <div className="flex justify-between items-start">
                      <h4 className="font-medium">{result.user.username || 'User'}</h4>
                      <span className={`text-xs px-2 py-1 rounded-full ${
                        result.post.type === 'mua' ? 'bg-blue-100 text-blue-800' : 'bg-green-100 text-green-800'
                      }`}>
                        {result.post.type === 'mua' ? 'Buying' : 'Selling'}
                      </span>
                    </div>
                    <p className="text-sm text-gray-800 mt-1">{result.post.content}</p>
                    <div className="mt-2 flex flex-wrap gap-1">
                      {result.post.category && (
                        <span className="text-xs bg-gray-100 px-2 py-1 rounded">
                          {result.post.category}
                        </span>
                      )}
                      {result.post.location && (
                        <span className="text-xs bg-gray-100 px-2 py-1 rounded">
                          📍 {result.post.location}
                        </span>
                      )}
                      {result.post.price > 0 && (
                        <span className="text-xs bg-gray-100 px-2 py-1 rounded">
                          💰 {result.post.price.toLocaleString()} VND
                        </span>
                      )}
                    </div>
                    <div className="mt-3 flex justify-between items-center">
                      <div className="text-xs text-gray-500">
                        Score: {Math.round(result.score * 100)}% match
                      </div>
                      {loadingItemId === result.post.id && (
                        <span className="text-xs text-primary flex items-center">
                          <div className="animate-spin mr-1 h-3 w-3 border border-primary border-t-transparent rounded-full"></div>
                          Creating chat...
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}