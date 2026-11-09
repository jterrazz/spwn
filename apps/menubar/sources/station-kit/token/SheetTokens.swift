import SwiftUI

// value: SPEC A5's sheet form: its rows on the 58 pt label grid, its
// fields 26 tall, the captions under them, the memory slider and the share
// chips, as the prototype draws them.
extension Token.Metric.Sheet {
    public static let field: CGFloat = 26
    public static let fieldRadius: CGFloat = 7
    public static let fieldPadding: CGFloat = 8
    public static let fieldGap: CGFloat = 6
    public static let caption: CGFloat = 4
    public static let rowGap: CGFloat = 10
    public static let popupGap: CGFloat = 5
    public static let chevron: CGFloat = 9
    public static let subtitleGap: CGFloat = 2
    public static let warnTop: CGFloat = 2

    public static let trackHeight: CGFloat = 4
    public static let knob: CGFloat = 18
    public static let tick = CGSize(width: 1.5, height: 4)
    public static let tickGap: CGFloat = 3
    public static let labelsGap: CGFloat = 3
    public static let helperTop: CGFloat = 6
    public static let hatch: CGFloat = 5
    public static let hatchStroke: CGFloat = 1.5

    public static let chip: CGFloat = 26
    public static let chipPadding: CGFloat = 8
    public static let chipGap: CGFloat = 6
    public static let box: CGFloat = 14
    public static let boxRadius: CGFloat = 4
    public static let boxStroke: CGFloat = 1
    public static let boxMark: CGFloat = 9
    public static let shareLabel: CGFloat = 52

    public static let splitDivider: CGFloat = 1
    public static let splitArrow: CGFloat = 26
    public static let footerHair: CGFloat = 0.5

    public static let shadow = (color: Color.black.opacity(0.38), radius: CGFloat(22), y: CGFloat(18))
    public static let shadowNear = (color: Color.black.opacity(0.18), radius: CGFloat(6), y: CGFloat(4))
    public static let blur: CGFloat = 1.5
}

extension Token.Role {
    public static let formLabel = Token.Role(12, .regular, .t2)
    public static let formValue = Token.Role(13, .regular, .t1)
    public static let formNote = Token.Role(13, .regular, .t3)
    public static let formCaption = Token.Role(11, .regular, .t3)
    public static let stopLabel = Token.Role(11, .regular, .t3)
    public static let stopPicked = Token.Role(11, .semibold, .t1)
}

extension Token.Symbol {
    public static let popup = "chevron.up.chevron.down"
}
