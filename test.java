import java.io.*;
import java.util.*;

interface IProduct {
    void setName(String name);
    String getName();
    void setCategory(String category);
    String getCategory();
    void setStock(int stock);
    int getStock();
    void setPrice(int price);
    int getPrice();
}

interface IInventory {
    void addProduct(IProduct product);
    void removeProduct(IProduct product);
    long calculateTotalValue();
    List<IProduct> getProductsByCategory(String category);
    List<IProduct> searchProductsByName(String name);
    Map<String, Integer> getProductsByCategoryWithCount();
    Map<String, List<IProduct>> getAllProductsByCategory();
}

class Product implements IProduct {
    
    private String name;
    private String category;
    private int stock;
    private int price;
    
    @Override
    public int getPrice() {
        // TODO Auto-generated method stub
        return price;
    }
    @Override
    public void setName(String name) {
        
        this.name = name;
    }
    @Override
    public void setCategory(String category) {
        this.category = category;
    }
    @Override
    public void setPrice(int price) {
        this.price = price;
    }
    @Override
    public void setStock(int stock) {
        this.stock = stock;
    }
    @Override
    public int getStock() {
        return stock;
    }
    @Override
    public String getCategory() {
        return category;
    }
    @Override
    public String getName() {
        return name;
    }
}

class Inventory implements IInventory {
    private List<IProduct> products = new ArrayList<>();
    @Override
    public void addProduct(IProduct product) {
        products.add(product);
    }
    @Override
    public long calculateTotalValue() {
        long total = 0;
        for (IProduct p : products) {
            total += (long) p.getPrice() * p.getStock();
        }
        return total;
    }
    @Override
    public Map<String, List<IProduct>> getAllProductsByCategory() {
        Map<String, List<IProduct>> map = new TreeMap<>();
        for (IProduct p : products) {
            map.computeIfAbsent(p.getCategory(), k -> new ArrayList<>()).add(p);
        }
        
        for(List<IProduct> list : map.values()) {
            list.sort(Comparator.comparing(IProduct::getName,String.CASE_INSENSITIVE_ORDER));
        }
        return map;
    }
    @Override
    public void removeProduct(IProduct product) {
        products.remove(product);
    }
    @Override
    public List<IProduct> searchProductsByName(String name) {
        List<IProduct> list = new ArrayList<>();
        for (IProduct p: products) {
            if (p.getName().equalsIgnoreCase(name)) {
                list.add(p);
            }
        }
        list.sort(Comparator.comparing(IProduct::getName,String.CASE_INSENSITIVE_ORDER));
        return list;
    }
    @Override
    public List<IProduct> getProductsByCategory(String category) {
        List<IProduct> list = new ArrayList<>();
        for (IProduct p: products) {
            if (p.getCategory().equalsIgnoreCase(category)) {
                list.add(p);
            }
        }
        
        list.sort(Comparator.comparing(IProduct::getName,String.CASE_INSENSITIVE_ORDER));
        return list;
    }
    @Override
    public Map<String, Integer> getProductsByCategoryWithCount() {
        Map<String, Integer> map = new TreeMap<>();
        for (IProduct p: products) {
            map.put(p.getCategory(), map.getOrDefault(p.getCategory(), 0) + 1);
        }
        return map;
    }
}
public class Solution {
    public static void main(String[] args) throws IOException {
        BufferedReader br = new BufferedReader(new InputStreamReader(System.in));
        PrintWriter out = new PrintWriter(System.out);

        IInventory inventory = new Inventory();
        int pCount = Integer.parseInt(br.readLine().trim());
        for (int i = 1; i <= pCount; i++) {
            String[] a = br.readLine().trim().split(" ");
            IProduct e = new Product();
            e.setName(a[0]);
            e.setCategory(a[1]);
            e.setStock(Integer.parseInt(a[2]));
            e.setPrice(Integer.parseInt(a[3]));
            inventory.addProduct(e);
        }
        String[] b = br.readLine().trim().split(" ");
        String randomCategoryName = b[0];
        String randomProductName = b[1];
        String productName = b[2];

        List<IProduct> getProductsByCategory = inventory.getProductsByCategory(randomCategoryName);

        out.println(randomCategoryName + ":");
        for (IProduct product : getProductsByCategory) {
            out.println("Product Name:" + product.getName() + " Category:" + product.getCategory());
        }

        List<IProduct> searchProductsByName = inventory.searchProductsByName(randomProductName);
        out.println(randomProductName + ":");
        for (IProduct product : searchProductsByName) {
            out.println("Product Name:" + product.getName() + " Category:" + product.getCategory());
        }
        out.println("Total Value:$" + inventory.calculateTotalValue());

        Map<String, Integer> getProductsByCategoryWithCount = inventory.getProductsByCategoryWithCount();
        for (Map.Entry<String, Integer> item : getProductsByCategoryWithCount.entrySet()) {
            out.println(item.getKey() + ":" + item.getValue());
        }

        Map<String, List<IProduct>> getAllProductsByCategory = inventory.getAllProductsByCategory();
        for (Map.Entry<String, List<IProduct>> item : getAllProductsByCategory.entrySet()) {
            out.println(item.getKey() + ":");
            for (IProduct item2 : item.getValue()) {
                out.println("Product Name:" + item2.getName() + " Price:" + item2.getPrice());
            }
        }

        List<IProduct> productsToDelete = inventory.searchProductsByName(productName);
        for (IProduct product : productsToDelete) {
            inventory.removeProduct(product);
        }
        out.println("New Total Value:$" + inventory.calculateTotalValue());

        out.flush();
        out.close();
    }
}