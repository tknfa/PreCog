import { useState, useEffect, useRef } from 'react'
import '../App.css'
import ProductCard from "../components/ProductCard"
import Sidebar from '../components/SideBar';
import ChatBox from '../components/ChatBox';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

const seedProducts = [
  {
    id: 1,
    name: "IPhone 16 Pro",
    price: "$899.99"
  },
  {
    id: 2,
    name: "Red Bull",
    price: "$2.99"
  },
  {
    id: 3,
    name: "Lamp",
    price: "$44.99"
  },
  {
    id: 4,
    name: "Bag of Chips",
    price: "$4.99"
  },
  {
    id: 5,
    name: "A Chair",
    price: "$149.99"
  }
];

const getProductName = (product) => product.name || product.title || "Recommended Item";
const getProductImage = (product) => product.image_url || product.imageUrl || product.img || "";

async function fetchAmazonProduct(product) {
  const query = encodeURIComponent(getProductName(product));
  const response = await fetch(`${API_BASE_URL}/amazon-search?query=${query}&limit=1`);

  if (!response.ok) {
    throw new Error(`Amazon search failed with status ${response.status}`);
  }

  const payload = await response.json();
  const [match] = Array.isArray(payload.items) ? payload.items : [];
  if (!match) {
    return product;
  }

  return {
    ...product,
    title: match.title || product.title || product.name,
    price: match.price || product.price,
    rating: match.rating ?? product.rating,
    image_url: match.image_url || product.image_url,
    amazon_link: match.amazon_link || product.amazon_link,
  };
}

function Home() {
  const [products, setProducts] = useState(seedProducts);
  const hasFetchedRecommendations = useRef(false);

  const getInitialBookmarks = () => {
    const stored = localStorage.getItem("bookmark");
    return stored ? JSON.parse(stored) : [];
  }

  const [bookmark, setBookmarks] = useState(getInitialBookmarks);

  useEffect(() => {
    localStorage.setItem("bookmark", JSON.stringify(bookmark));
  }, [bookmark]);

  useEffect(() => {
    if (hasFetchedRecommendations.current) {
      return;
    }
    hasFetchedRecommendations.current = true;

    let isCancelled = false;

    const loadRecommendationImages = async () => {
      const enrichedProducts = [];

      for (const product of seedProducts) {
        try {
          const enrichedProduct = await fetchAmazonProduct(product);
          enrichedProducts.push(enrichedProduct);
        } catch (error) {
          console.error(`Failed to enrich recommendation (${getProductName(product)}):`, error);
          enrichedProducts.push(product);
        }

        if (isCancelled) {
          return;
        }
      }

      setProducts(enrichedProducts);
      setBookmarks((prev) =>
        prev.map((item) => {
          const match = enrichedProducts.find((candidate) => candidate.id === item.id);
          return match ? { ...item, ...match } : item;
        }),
      );
    };

    loadRecommendationImages();

    return () => {
      isCancelled = true;
    };
  }, []);

  const addToBookmark = (product) => {
    setBookmarks((prev) => {
      if (prev.find((item) => item.id === product.id)) {
        return prev;
      }
      return [...prev, product];
    });
  };

  const removeFromBookmark = (id) => {
    setBookmarks((prev) => prev.filter((item) => item.id !== id));
  }

  return (
    <div className="layout">
      <div className="main">
        <div className="mainBox">
          <div className="product-slider">
            {products.map((product) => (
              <ProductCard
                key={product.id}
                name={getProductName(product)}
                img={getProductImage(product)}
                imageUrl={product.image_url || product.imageUrl}
                price={product.price}
                isBookmarked={false}
                onBookmark={() => addToBookmark(product)}
              />
            ))}
          </div>
        </div>

        <ChatBox/>

        <div className="subBox">
          <h2>Your Bookmarks</h2>
          <div className="bookmark-container">
            {bookmark.length === 0 ? (
              <p>No items added yet.</p>
            ) : (
              <div className="product-slider">
                {bookmark.map((item) => (
                  <ProductCard
                    key={item.id}
                    name={getProductName(item)}
                    img={getProductImage(item)}
                    imageUrl={item.image_url || item.imageUrl}
                    price={item.price}
                    isBookmarked={true}
                    onBookmark={() => removeFromBookmark(item.id)}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      <Sidebar/>
    </div>
  );
}

export default Home;
