import numpy as np
import matplotlib.pyplot as plt
import random

class GalaxyGenerator:
    def __init__(self, num_stars=10000, radius_ly=50000):
        self.num_stars = num_stars
        self.radius_ly = radius_ly # Radius der Galaxie in Lichtjahren
        self.stars = []
        
        # 1. Spektralklassen-Wahrscheinlichkeiten (Realismus-Ansatz)
        # Typen: O, B, A, F, G, K, M
        self.spectral_types = ['O', 'B', 'A', 'F', 'G', 'K', 'M']
        # Prozentuale Verteilung (ca. Milchstraße)
        self.spectral_probs = [0.00003, 0.0013, 0.006, 0.03, 0.076, 0.121, 0.76567]
        
        # Farben für die Visualisierung (Hex)
        self.type_colors = {
            'O': '#9db4ff', 'B': '#aabfff', 'A': '#cad8ff', 
            'F': '#fbf8ff', 'G': '#fff4e8', 'K': '#ffddb4', 'M': '#ffbd6f'
        }

    def generate_star_properties(self):
        """Weist einem Stern basierend auf Wahrscheinlichkeiten eine Klasse zu."""
        s_type = np.random.choice(self.spectral_types, p=self.spectral_probs)
        return s_type, self.type_colors[s_type]

    def generate_positions(self, arms=2, winding=2.0, spread=0.5):
        """
        Erzeugt Positionen für eine Balkenspiralgalaxie (SB-Typ).
        arms: Anzahl der Spiralarme
        winding: Wie stark sich die Arme drehen
        spread: Wie "ausgefranst" die Arme sind
        """
        positions = []
        colors = []
        types = []

        # Wir teilen die Sterne auf Komponenten auf
        # Kern (Bulge) & Balken: ca. 20%
        # Scheibe (Disk): ca. 80%
        num_core = int(self.num_stars * 0.10)
        num_bar = int(self.num_stars * 0.10)
        num_disk = self.num_stars - num_core - num_bar

        # --- A. KERNE (BULGE) ---
        # Kugelförmige Normalverteilung um das Zentrum
        for _ in range(num_core):
            # Zufällige Richtung
            theta = random.uniform(0, 2 * np.pi)
            phi = random.uniform(0, np.pi)
            # Radius: Dicht im Zentrum, schnell abfallend
            r = abs(np.random.normal(0, self.radius_ly * 0.15)) 
            
            x = r * np.sin(phi) * np.cos(theta)
            y = r * np.sin(phi) * np.sin(theta)
            z = r * np.cos(phi) * 0.6 # Etwas abgeflacht
            
            s_type, s_color = self.generate_star_properties()
            positions.append([x, y, z])
            colors.append(s_color)
            types.append(s_type)

        # --- B. BALKEN (BAR) ---
        # Längliche Box oder Ellipsoid im Zentrum
        bar_length = self.radius_ly * 0.4
        bar_width = self.radius_ly * 0.1
        bar_height = self.radius_ly * 0.05
        
        for _ in range(num_bar):
            # Einfache Box-Verteilung für den Balken (rotiert um 45 Grad optional)
            x = np.random.normal(0, bar_length / 2)
            y = np.random.normal(0, bar_width / 2)
            z = np.random.normal(0, bar_height / 2)
            
            s_type, s_color = self.generate_star_properties()
            positions.append([x, y, z])
            colors.append(s_color)
            types.append(s_type)

        # --- C. SCHEIBE & SPIRALARME ---
        for _ in range(num_disk):
            # 1. Radius: Exponentieller Abfall der Dichte nach außen
            # Wir wählen einen Radius und skalieren ihn
            r_norm = np.random.random() ** 1.5 # Sorgt für mehr Dichte innen
            r = r_norm * self.radius_ly
            
            # Mindestradius, damit sie nicht alle im Kern sitzen
            if r < self.radius_ly * 0.2: 
                r += self.radius_ly * 0.2

            # 2. Winkel (Basis-Spirale)
            # Logarithmische Spirale Formel-Annäherung
            # Winkel nimmt mit Radius zu
            theta = winding * r_norm * np.pi 
            
            # 3. Arm-Zuordnung
            # Verschiebe den Stern zu einem der N Arme (z.B. 0 oder 180 Grad)
            arm_offset = (np.random.randint(0, arms) * 2 * np.pi / arms)
            theta += arm_offset

            # 4. Streuung (Scatter)
            # Sterne liegen nicht perfekt auf der Linie, sondern streuen darum
            theta += np.random.normal(0, spread)

            # 5. Konvertierung Polar -> Kartesisch
            x = r * np.cos(theta)
            y = r * np.sin(theta)
            
            # 6. Höhe (Z-Achse) - Die Scheibe ist flach
            z = np.random.normal(0, self.radius_ly * 0.02)

            s_type, s_color = self.generate_star_properties()
            positions.append([x, y, z])
            colors.append(s_color)
            types.append(s_type)

        return np.array(positions), colors, types

    def visualize(self):
        """Erstellt einen 2D Plot (Draufsicht) zur Überprüfung."""
        pos, colors, _ = self.generate_positions()
        
        plt.figure(figsize=(10, 10))
        plt.style.use('dark_background')
        
        # X und Y Koordinaten plotten
        # s=1 ist die Punktgröße, alpha=0.6 macht sie leicht transparent
        plt.scatter(pos[:, 0], pos[:, 1], c=colors, s=1, alpha=0.7)
        
        plt.title(f"Generierte Balkenspiralgalaxie ({self.num_stars} Sterne)")
        plt.axis('equal') # Wichtig, damit Kreise rund sind
        plt.show()
        
    def get_data(self):
        """Gibt die Rohdaten für das Spiel zurück."""
        pos, colors, types = self.generate_positions()
        game_data = []
        for i in range(len(pos)):
            game_data.append({
                "id": i,
                "x": pos[i][0],
                "y": pos[i][1],
                "z": pos[i][2],
                "type": types[i],
                "color": colors[i]
            })
        return game_data

# --- AUSFÜHRUNG ---
if __name__ == "__main__":
    # Erstelle eine Galaxie mit 20.000 Sternen
    galaxy = GalaxyGenerator(num_stars=20000)
    
    # Zeige das Bild an
    galaxy.visualize()
    
    # Beispiel: Daten für das erste Sternsystem abrufen
    data = galaxy.get_data()
    print("Beispiel-Sternsystem:", data[0])
